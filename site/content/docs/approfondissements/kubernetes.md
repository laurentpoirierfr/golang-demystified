---
title: "Kubernetes : écrire un opérateur"
linkTitle: "15. Kubernetes"
weight: 150
---

[Kubernetes](https://kubernetes.io) est écrit en Go, et c'est en Go qu'on
l'étend. Ce chapitre présente les outils de l'écosystème, puis construit un
**opérateur** : un programme qui apprend à Kubernetes à gérer une application,
ici l'API des tâches du chapitre « Une API web complète ».

Le code est dans [`examples/approfondissements/operateur`](https://github.com/laurentpoirierfr/golang-demystified/tree/main/examples/approfondissements/operateur).

## Le principe : la boucle de réconciliation

Kubernetes fonctionne de manière **déclarative**. On ne lui dit pas « démarre
deux conteneurs », mais « je veux deux conteneurs ». Des **contrôleurs**
comparent en permanence l'état souhaité, décrit dans des ressources (un
`Deployment`, un `Service`…), à l'état réel de la grappe, et agissent pour les
rapprocher. C'est la **boucle de réconciliation** :

```
        ┌────────────────────────────────────────────┐
        │                                            │
        ▼                                            │
  état souhaité ──► comparer ──► agir ──► état réel ─┘
   (ressource)                 (créer, modifier, supprimer)
```

Un **opérateur** applique ce principe à une application métier. On définit
une **ressource personnalisée** (*Custom Resource*), qui décrit l'application
dans son vocabulaire, et un contrôleur qui sait la mettre en œuvre :

```yaml
apiVersion: taches.example.com/v1alpha1
kind: ApiTaches
metadata:
  name: taches
  namespace: demo
spec:
  image: ghcr.io/exemple/taches:1.0
  replicas: 2
  base: postgres://gopher:secret@postgres:5432/taches
```

À partir de cette ressource, l'opérateur crée et maintient un `Deployment`
(les conteneurs de l'API) et un `Service` (son adresse réseau), puis publie
l'état observé dans le statut de la ressource :

```
$ kubectl get apitaches -n demo
NAME     REPLICAS   PRÊTES   ÂGE
taches   2          2        3m
```

Si quelqu'un modifie le `Deployment` à la main, l'opérateur rétablit l'état
souhaité. Si l'on supprime l'`ApiTaches`, Kubernetes supprime ce qu'elle
possède. Les opérateurs servent à automatiser ce qu'un administrateur ferait à
la main : installer une base de données en grappe, la sauvegarder, gérer ses
mises à jour, ses certificats…

## Les outils de l'écosystème

| Outil | Rôle |
|-------|------|
| [client-go](https://github.com/kubernetes/client-go) | La bibliothèque cliente officielle : accès à l'API, *informers* (caches tenus à jour par surveillance), files de travail |
| [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) | Construit au-dessus de client-go : gestionnaire, contrôleurs, client unifié, élection d'un chef, webhooks. La base de presque tous les opérateurs |
| [controller-tools](https://github.com/kubernetes-sigs/controller-tools) (`controller-gen`) | Génère, à partir des types Go et de marqueurs en commentaires, le code `DeepCopy`, les CRD et les rôles RBAC |
| [Kubebuilder](https://book.kubebuilder.io) | Génère la structure complète d'un projet d'opérateur (types, contrôleur, tests, Makefile, manifestes) |
| [Operator SDK](https://sdk.operatorframework.io) | Construit sur Kubebuilder, ajoute l'intégration à l'*Operator Lifecycle Manager* et la publication sur OperatorHub |
| [envtest](https://book.kubebuilder.io/reference/envtest) | Démarre un vrai serveur d'API Kubernetes et etcd, sans nœuds, pour les tests d'intégration |
| [kind](https://kind.sigs.k8s.io) | Une grappe Kubernetes complète dans des conteneurs Docker ou Podman, pour le développement |

La plupart des projets démarrent avec Kubebuilder, qui génère le squelette :

```bash
kubebuilder init --domain example.com --repo github.com/utilisateur/operateur-taches
kubebuilder create api --group taches --version v1alpha1 --kind ApiTaches
```

L'exemple de ce chapitre utilise directement les mêmes bibliothèques, sans
le squelette, pour que chaque élément soit visible.

```
operateur/
├── api/v1alpha1/                   la ressource personnalisée (types Go)
│   ├── apitaches_types.go
│   ├── groupversion.go
│   └── zz_generated.deepcopy.go    généré par controller-gen
├── internal/controleur/            la boucle de réconciliation
├── cmd/operateur/                  le gestionnaire
├── config/
│   ├── crd/                        la CRD, générée par controller-gen
│   ├── rbac/                       le rôle RBAC, généré par controller-gen
│   └── exemples/                   une ApiTaches d'exemple
└── generate.go                     les directives go:generate
```

## La ressource personnalisée

Une ressource personnalisée est définie par des **types Go**. La partie
`Spec` décrit l'état souhaité, écrite par l'utilisateur ; la partie `Status`
décrit l'état observé, écrite par l'opérateur. Les **marqueurs**
`+kubebuilder:…` déclarent la validation, les valeurs par défaut et les
colonnes affichées par `kubectl` :

```go
// ApiTachesSpec décrit l'état souhaité : ce que l'utilisateur demande.
type ApiTachesSpec struct {
	// Image est l'image de conteneur de l'API des tâches.
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`

	// Replicas est le nombre d'instances de l'API.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	// +kubebuilder:default=1
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Base est la chaîne de connexion à la base (TACHES_BASE).
	// +optional
	Base string `json:"base,omitempty"`
}

// ApiTachesStatus décrit l'état observé : ce que l'opérateur constate.
type ApiTachesStatus struct {
	ReplicasPretes int32              `json:"replicasPretes,omitempty"`
	Conditions     []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Prêtes",type=integer,JSONPath=`.status.replicasPretes`
type ApiTaches struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiTachesSpec   `json:"spec"`
	Status ApiTachesStatus `json:"status,omitempty"`
}
```

Le statut utilise les **conditions** standard de Kubernetes
(`metav1.Condition`) : une liste d'états nommés (`Pret`), chacun avec une
valeur, une raison et un message. Les outils comme `kubectl wait` savent les
lire :

```bash
kubectl wait --for=condition=Pret apitaches/taches -n demo
```

Le package d'API ne dépend que de `k8s.io/apimachinery` : d'autres projets,
comme des clients ou des outils, peuvent l'importer sans tirer tout
controller-runtime.

### Générer le code et les manifestes

Kubernetes exige que chaque objet sache se copier en profondeur (méthode
`DeepCopyObject`). Écrire ces méthodes à la main serait fastidieux et
fragile : `controller-gen` les génère, ainsi que la **CRD**, le manifeste
YAML qui déclare la nouvelle ressource à la grappe. Il est déclaré comme
outil du module, avec la directive `tool` de Go 1.24 :

```bash
go get -tool sigs.k8s.io/controller-tools/cmd/controller-gen@v0.23.0
```

```go
//go:generate go tool controller-gen object paths=./api/...
//go:generate go tool controller-gen crd paths=./api/... output:crd:artifacts:config=config/crd
//go:generate go tool controller-gen rbac:roleName=operateur-taches paths=./internal/... output:rbac:artifacts:config=config/rbac
```

La CRD générée contient le schéma de validation tiré des types et des
marqueurs. Le serveur d'API refuse donc une `ApiTaches` avec 20 replicas ou
sans image, avant même que l'opérateur ne la voie :

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: apitaches.taches.example.com
spec:
  group: taches.example.com
  names:
    kind: ApiTaches
    plural: apitaches
  scope: Namespaced
  versions:
  - name: v1alpha1
    additionalPrinterColumns:
    - jsonPath: .spec.replicas
      name: Replicas
      type: integer
    - jsonPath: .status.replicasPretes
      name: Prêtes
      type: integer
    …
```

## Le contrôleur

### La réconciliation

La fonction `Reconcile` est appelée à chaque changement de la ressource, ou
d'un objet qu'elle possède. Elle reçoit seulement le **nom** de la ressource,
pas la nature du changement : elle relit l'état souhaité et refait tout le
rapprochement. Elle doit donc être **idempotente** : appelée dix fois de
suite, elle produit le même résultat qu'une seule fois.

```go
func (r *ApiTachesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// 1. Lire l'état souhaité.
	var api tachesv1.ApiTaches
	if err := r.Get(ctx, req.NamespacedName, &api); err != nil {
		// Supprimée entre-temps : ses objets seront supprimés par Kubernetes,
		// grâce aux références de propriétaire. Il n'y a rien à faire.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Rapprocher le Deployment de l'état souhaité.
	deploiement := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: api.Name, Namespace: api.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deploiement, func() error {
		r.decritDeploiement(&api, deploiement)
		return controllerutil.SetControllerReference(&api, deploiement, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("deployment : %w", err)
	}

	// 3. Rapprocher le Service, de la même façon.
	…

	// 4. Publier l'état observé dans le statut.
	api.Status.ReplicasPretes = deploiement.Status.ReadyReplicas
	pret := metav1.Condition{
		Type:    ConditionPret,
		Status:  metav1.ConditionFalse,
		Reason:  "Demarrage",
		Message: fmt.Sprintf("%d instance(s) prête(s) sur %d", deploiement.Status.ReadyReplicas, api.Spec.Replicas),
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
```

Trois fonctions de controller-runtime font l'essentiel du travail :

- **`CreateOrUpdate`** lit l'objet, applique la fonction de description, puis
  le crée s'il n'existe pas ou le met à jour s'il a changé. La fonction de
  description ne modifie que les champs gérés par l'opérateur : ceux que
  Kubernetes ajoute lui-même sont conservés ;
- **`SetControllerReference`** fait de l'`ApiTaches` la **propriétaire** du
  `Deployment`. Le ramasse-miettes de Kubernetes supprimera le `Deployment`
  avec elle, et toute modification du `Deployment` déclenchera une
  réconciliation ;
- **`client.IgnoreNotFound`** traite le cas, normal, d'une ressource supprimée
  entre l'événement et la réconciliation.

Une erreur renvoyée par `Reconcile` n'est pas fatale : la ressource est remise
dans la file de travail, et retraitée avec un délai croissant.

### Décrire l'état souhaité

La description du `Deployment` traduit le vocabulaire de l'application en
objets Kubernetes. Elle configure notamment la **sonde de disponibilité** sur
le contrôle de santé `/sante` de l'API : Kubernetes n'envoie du trafic qu'aux
instances qui répondent.

```go
func (r *ApiTachesReconciler) decritDeploiement(api *tachesv1.ApiTaches, d *appsv1.Deployment) {
	etiquettes := map[string]string{"app.kubernetes.io/name": "api-taches", "app.kubernetes.io/instance": api.Name}
	replicas := api.Spec.Replicas
	d.Spec.Replicas = &replicas
	d.Spec.Selector = &metav1.LabelSelector{MatchLabels: etiquettes}
	d.Spec.Template.Labels = etiquettes
	d.Spec.Template.Spec.Containers = []corev1.Container{{
		Name:  "api",
		Image: api.Spec.Image,
		Env:   []corev1.EnvVar{{Name: "TACHES_BASE", Value: api.Spec.Base}},
		Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}},
		ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{Path: "/sante", Port: intstr.FromString("http")},
		}},
	}}
}
```

### Ce que l'on surveille

`SetupWithManager` déclare la ressource principale et les objets qu'elle
possède. Un changement sur l'un d'eux déclenche la réconciliation de
l'`ApiTaches` propriétaire :

```go
func (r *ApiTachesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tachesv1.ApiTaches{}).  // la ressource principale
		Owns(&appsv1.Deployment{}). // et les objets qu'elle possède
		Owns(&corev1.Service{}).
		Complete(r)
}
```

### Les droits : RBAC

Un opérateur s'exécute avec un compte de service, qui n'a que les droits
qu'on lui accorde. Ces droits sont déclarés par des marqueurs, à côté du code
qui les utilise, et `controller-gen` en génère le rôle :

```go
// +kubebuilder:rbac:groups=taches.example.com,resources=apitaches,verbs=get;list;watch
// +kubebuilder:rbac:groups=taches.example.com,resources=apitaches/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch
```

L'opérateur peut créer des `Deployment`, mais pas les supprimer : c'est le
ramasse-miettes de Kubernetes qui s'en charge, grâce aux références de
propriétaire. Le principe du moindre privilège s'applique aussi aux
opérateurs.

## Le gestionnaire

Le programme principal crée un **gestionnaire** (*manager*), qui se connecte à
la grappe, partage un cache des objets surveillés entre les contrôleurs, expose
les sondes de santé, et gère l'**élection d'un chef** : plusieurs copies de
l'opérateur peuvent tourner pour la haute disponibilité, mais une seule agit à
la fois.

```go
func main() {
	ctrl.SetLogger(zap.New())

	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme) // Deployment, Service…
	tachesv1.AddToScheme(scheme)        // ApiTaches

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: ":8081",
		LeaderElection:         true,
		LeaderElectionID:       "operateur-taches.taches.example.com",
	})
	…
	r := &controleur.ApiTachesReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
	if err := r.SetupWithManager(mgr); err != nil {
		…
	}
	mgr.Start(ctrl.SetupSignalHandler()) // jusqu'à SIGTERM
}
```

Le **scheme** associe chaque type Go à son groupe, sa version et son type
Kubernetes : c'est ce qui permet au client de savoir que `*ApiTaches`
correspond à `taches.example.com/v1alpha1, Kind=ApiTaches`.

`ctrl.GetConfigOrDie()` trouve la configuration de connexion : le fichier
`~/.kube/config` en développement, ou le compte de service quand l'opérateur
s'exécute dans la grappe.

## Tester l'opérateur

### Avec un client factice

controller-runtime fournit un **client factice**, en mémoire, qui se comporte
comme le serveur d'API pour les opérations courantes. La boucle de
réconciliation se teste ainsi sans grappe, en quelques millisecondes :

```go
func banc(t *testing.T) (*ApiTachesReconciler, client.Client, ctrl.Request) {
	s := runtime.NewScheme()
	clientgoscheme.AddToScheme(s)
	tachesv1.AddToScheme(s)

	api := &tachesv1.ApiTaches{
		ObjectMeta: metav1.ObjectMeta{Name: "taches", Namespace: "demo"},
		Spec:       tachesv1.ApiTachesSpec{Image: "ghcr.io/exemple/taches:1.0", Replicas: 2},
	}
	c := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(api).
		WithStatusSubresource(&tachesv1.ApiTaches{}, &appsv1.Deployment{}).
		Build()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "taches", Namespace: "demo"}}
	return &ApiTachesReconciler{Client: c, Scheme: s}, c, req
}
```

Les tests vérifient les comportements attendus d'un opérateur :

| Test | Comportement vérifié |
|------|----------------------|
| `TestCreeDeploymentEtService` | le `Deployment` et le `Service` sont créés, avec la bonne image, les bons replicas, et l'`ApiTaches` pour propriétaire |
| `TestSuitLesChangements` | un changement de `replicas` est appliqué ; une modification manuelle du `Deployment` est **annulée** |
| `TestStatut` | la condition `Pret` passe à vrai quand Kubernetes signale les instances prêtes |
| `TestRessourceSupprimee` | une ressource supprimée n'est pas une erreur |
| `TestIdempotence` | trois réconciliations successives ne créent qu'un seul `Deployment` |

```go
// Quelqu'un modifie le Deployment à la main : l'opérateur rétablit l'état souhaité.
d.Spec.Template.Spec.Containers[0].Image = "pirate:latest"
c.Update(ctx, &d)
r.Reconcile(ctx, req)
c.Get(ctx, req.NamespacedName, &d)
if got := d.Spec.Template.Spec.Containers[0].Image; got != "ghcr.io/exemple/taches:1.0" {
	t.Errorf("image = %s, l'opérateur aurait dû la rétablir", got)
}
```

### Avec envtest et kind

Le client factice ne reproduit pas tout : la validation de la CRD, les
valeurs par défaut, le ramasse-miettes. Deux outils complètent les tests :

- **envtest** démarre un vrai serveur d'API et etcd, sans nœuds : la CRD est
  réellement installée et validée. C'est l'outil de test des projets
  Kubebuilder ;
- **kind** fournit une grappe complète pour les tests de bout en bout, où les
  conteneurs s'exécutent vraiment.

```bash
kind create cluster --name taches
kubectl apply -f config/crd/ -f config/rbac/
go run ./cmd/operateur &                       # l'opérateur, hors de la grappe
kubectl create namespace demo
kubectl apply -f config/exemples/apitaches.yaml
kubectl get apitaches,deployments,services -n demo
```

## Aller plus loin

Un opérateur de production ajoute en général :

- des **finaliseurs**, pour nettoyer des ressources externes (une sauvegarde,
  un enregistrement DNS) avant la suppression de la ressource ;
- des **webhooks** de validation et de conversion, pour des règles que le
  schéma ne peut pas exprimer, et pour faire évoluer la ressource de
  `v1alpha1` vers `v1` ;
- des **événements** Kubernetes (`kubectl describe` les affiche) et des
  **métriques** Prometheus, exposées par le gestionnaire ;
- la **mise à jour par patch** (*Server-Side Apply*) plutôt que par
  `Update`, qui évite les conflits quand plusieurs acteurs modifient un même
  objet.

Kubebuilder génère le squelette de chacun de ces éléments, et son
[livre en ligne](https://book.kubebuilder.io) est la référence pour aller plus
loin.
