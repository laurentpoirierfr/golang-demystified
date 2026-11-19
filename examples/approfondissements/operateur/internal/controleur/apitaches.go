// Package controleur contient le contrôleur de la ressource ApiTaches : la
// boucle de réconciliation qui rapproche l'état réel de la grappe de l'état
// souhaité décrit par l'utilisateur.
package controleur

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	tachesv1 "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/operateur/api/v1alpha1"
)

// ConditionPret est le type de la condition qui indique si l'API est prête.
const ConditionPret = "Pret"

// ApiTachesReconciler réconcilie les ressources ApiTaches.
type ApiTachesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Les marqueurs ci-dessous décrivent les droits dont l'opérateur a besoin ;
// controller-gen en génère le rôle RBAC (config/rbac).
//
// +kubebuilder:rbac:groups=taches.example.com,resources=apitaches,verbs=get;list;watch
// +kubebuilder:rbac:groups=taches.example.com,resources=apitaches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch

// Reconcile est appelée à chaque changement d'une ApiTaches, ou d'un objet
// qu'elle possède (Deployment, Service). Elle doit être idempotente : appelée
// dix fois de suite, elle produit le même résultat qu'une seule fois.
func (r *ApiTachesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	journal := log.FromContext(ctx)

	// 1. Lire l'état souhaité.
	var api tachesv1.ApiTaches
	if err := r.Get(ctx, req.NamespacedName, &api); err != nil {
		// Supprimée entre-temps : ses objets seront supprimés par Kubernetes,
		// grâce aux références de propriétaire. Il n'y a rien à faire.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Rapprocher le Deployment de l'état souhaité.
	deploiement := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: api.Name, Namespace: api.Namespace}}
	operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploiement, func() error {
		r.decritDeploiement(&api, deploiement)
		// L'ApiTaches devient propriétaire du Deployment : supprimer l'une
		// supprime l'autre, et toute modification du Deployment déclenche
		// une réconciliation.
		return controllerutil.SetControllerReference(&api, deploiement, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("deployment : %w", err)
	}
	if operation != controllerutil.OperationResultNone {
		journal.Info("deployment réconcilié", "operation", operation)
	}

	// 3. Rapprocher le Service.
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: api.Name, Namespace: api.Namespace}}
	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		r.decritService(&api, service)
		return controllerutil.SetControllerReference(&api, service, r.Scheme)
	}); err != nil {
		return ctrl.Result{}, fmt.Errorf("service : %w", err)
	}

	// 4. Publier l'état observé dans le statut.
	api.Status.ReplicasPretes = deploiement.Status.ReadyReplicas
	pret := metav1.Condition{
		Type:               ConditionPret,
		Status:             metav1.ConditionFalse,
		Reason:             "Demarrage",
		Message:            fmt.Sprintf("%d instance(s) prête(s) sur %d", deploiement.Status.ReadyReplicas, api.Spec.Replicas),
		ObservedGeneration: api.Generation,
	}
	if deploiement.Status.ReadyReplicas >= api.Spec.Replicas {
		pret.Status, pret.Reason = metav1.ConditionTrue, "InstancesPretes"
	}
	meta.SetStatusCondition(&api.Status.Conditions, pret)
	if err := r.Status().Update(ctx, &api); err != nil {
		if apierrors.IsConflict(err) {
			// L'objet a changé depuis sa lecture : on recommence un peu plus
			// tard, avec la version à jour.
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{}, fmt.Errorf("statut : %w", err)
	}
	return ctrl.Result{}, nil
}

// decritDeploiement applique l'état souhaité au Deployment. Seuls les champs
// gérés par l'opérateur sont modifiés : ceux ajoutés par Kubernetes sont
// conservés.
func (r *ApiTachesReconciler) decritDeploiement(api *tachesv1.ApiTaches, d *appsv1.Deployment) {
	etiquettes := map[string]string{"app.kubernetes.io/name": "api-taches", "app.kubernetes.io/instance": api.Name}
	replicas := api.Spec.Replicas
	if replicas == 0 {
		replicas = 1
	}
	d.Spec.Replicas = &replicas
	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: etiquettes}
	d.Spec.Template.Labels = etiquettes

	var env []corev1.EnvVar
	if api.Spec.Base != "" {
		env = append(env, corev1.EnvVar{Name: "TACHES_BASE", Value: api.Spec.Base})
	}
	d.Spec.Template.Spec.Containers = []corev1.Container{{
		Name:  "api",
		Image: api.Spec.Image,
		Env:   env,
		Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}},
		// L'API expose /sante : Kubernetes n'envoie de trafic qu'aux
		// instances qui répondent.
		ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{Path: "/sante", Port: intstr.FromString("http")},
		}},
	}}
}

func (r *ApiTachesReconciler) decritService(api *tachesv1.ApiTaches, s *corev1.Service) {
	s.Spec.Selector = map[string]string{"app.kubernetes.io/name": "api-taches", "app.kubernetes.io/instance": api.Name}
	s.Spec.Ports = []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromString("http")}}
}

// SetupWithManager déclare au gestionnaire les ressources surveillées.
func (r *ApiTachesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tachesv1.ApiTaches{}). // la ressource principale
		Owns(&appsv1.Deployment{}). // et les objets qu'elle possède
		Owns(&corev1.Service{}).
		Complete(r)
}
