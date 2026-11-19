package controleur

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	tachesv1 "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/operateur/api/v1alpha1"
)

// banc prépare un client Kubernetes factice, en mémoire, qui contient une
// ApiTaches : le contrôleur se teste sans grappe.
func banc(t *testing.T) (*ApiTachesReconciler, client.Client, ctrl.Request) {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil { // Deployment, Service…
		t.Fatal(err)
	}
	if err := tachesv1.AddToScheme(s); err != nil { // ApiTaches
		t.Fatal(err)
	}
	api := &tachesv1.ApiTaches{
		ObjectMeta: metav1.ObjectMeta{Name: "taches", Namespace: "demo"},
		Spec:       tachesv1.ApiTachesSpec{Image: "ghcr.io/exemple/taches:1.0", Replicas: 2, Base: "postgres://db/taches"},
	}
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(api).
		WithStatusSubresource(&tachesv1.ApiTaches{}, &appsv1.Deployment{}).
		Build()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "taches", Namespace: "demo"}}
	return &ApiTachesReconciler{Client: c, Scheme: s}, c, req
}

func TestCreeDeploymentEtService(t *testing.T) {
	r, c, req := banc(t)
	if _, err := r.Reconcile(t.Context(), req); err != nil {
		t.Fatal(err)
	}

	var d appsv1.Deployment
	if err := c.Get(t.Context(), req.NamespacedName, &d); err != nil {
		t.Fatalf("Deployment absent : %v", err)
	}
	conteneur := d.Spec.Template.Spec.Containers[0]
	if *d.Spec.Replicas != 2 || conteneur.Image != "ghcr.io/exemple/taches:1.0" {
		t.Errorf("Deployment inattendu : %d replicas, image %s", *d.Spec.Replicas, conteneur.Image)
	}
	if conteneur.Env[0].Name != "TACHES_BASE" || conteneur.Env[0].Value != "postgres://db/taches" {
		t.Errorf("variable TACHES_BASE absente : %+v", conteneur.Env)
	}
	if len(d.OwnerReferences) != 1 || d.OwnerReferences[0].Kind != "ApiTaches" {
		t.Errorf("le Deployment devrait appartenir à l'ApiTaches : %+v", d.OwnerReferences)
	}

	var s corev1.Service
	if err := c.Get(t.Context(), req.NamespacedName, &s); err != nil {
		t.Fatalf("Service absent : %v", err)
	}
	if s.Spec.Ports[0].Port != 80 {
		t.Errorf("port du Service = %d", s.Spec.Ports[0].Port)
	}
}

func TestSuitLesChangements(t *testing.T) {
	r, c, req := banc(t)
	ctx := t.Context()
	r.Reconcile(ctx, req)

	// L'utilisateur passe à 5 instances.
	var api tachesv1.ApiTaches
	c.Get(ctx, req.NamespacedName, &api)
	api.Spec.Replicas = 5
	if err := c.Update(ctx, &api); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Reconcile(ctx, req); err != nil {
		t.Fatal(err)
	}
	var d appsv1.Deployment
	c.Get(ctx, req.NamespacedName, &d)
	if *d.Spec.Replicas != 5 {
		t.Errorf("replicas = %d, attendu 5", *d.Spec.Replicas)
	}

	// Quelqu'un modifie le Deployment à la main : l'opérateur rétablit l'état souhaité.
	d.Spec.Template.Spec.Containers[0].Image = "pirate:latest"
	c.Update(ctx, &d)
	r.Reconcile(ctx, req)
	c.Get(ctx, req.NamespacedName, &d)
	if got := d.Spec.Template.Spec.Containers[0].Image; got != "ghcr.io/exemple/taches:1.0" {
		t.Errorf("image = %s, l'opérateur aurait dû la rétablir", got)
	}
}

func TestStatut(t *testing.T) {
	r, c, req := banc(t)
	ctx := t.Context()
	r.Reconcile(ctx, req)

	statut := func() (int32, *metav1.Condition) {
		var api tachesv1.ApiTaches
		c.Get(ctx, req.NamespacedName, &api)
		return api.Status.ReplicasPretes, meta.FindStatusCondition(api.Status.Conditions, ConditionPret)
	}
	if pretes, cond := statut(); pretes != 0 || cond == nil || cond.Status != metav1.ConditionFalse {
		t.Fatalf("au démarrage : %d prêtes, condition %+v", pretes, cond)
	}

	// Kubernetes signale que les deux instances sont prêtes.
	var d appsv1.Deployment
	c.Get(ctx, req.NamespacedName, &d)
	d.Status.ReadyReplicas = 2
	if err := c.Status().Update(ctx, &d); err != nil {
		t.Fatal(err)
	}
	r.Reconcile(ctx, req)
	if pretes, cond := statut(); pretes != 2 || cond.Status != metav1.ConditionTrue || cond.Message != "2 instance(s) prête(s) sur 2" {
		t.Errorf("une fois prêt : %d prêtes, condition %+v", pretes, cond)
	}
}

func TestRessourceSupprimee(t *testing.T) {
	r, _, _ := banc(t)
	absente := ctrl.Request{NamespacedName: types.NamespacedName{Name: "absente", Namespace: "demo"}}
	if res, err := r.Reconcile(t.Context(), absente); err != nil || res.RequeueAfter != 0 {
		t.Errorf("Reconcile d'une ressource supprimée = %+v, %v", res, err)
	}
}

func TestIdempotence(t *testing.T) {
	r, c, req := banc(t)
	for range 3 {
		if _, err := r.Reconcile(t.Context(), req); err != nil {
			t.Fatal(err)
		}
	}
	var liste appsv1.DeploymentList
	c.List(t.Context(), &liste, client.InNamespace("demo"))
	if len(liste.Items) != 1 {
		t.Errorf("%d Deployments après trois réconciliations, attendu 1", len(liste.Items))
	}
}
