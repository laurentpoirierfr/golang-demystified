// Command operateur démarre l'opérateur de l'API des tâches. Il se connecte à
// la grappe décrite par ~/.kube/config (ou au compte de service, s'il
// s'exécute dans la grappe), et réconcilie les ressources ApiTaches.
package main

import (
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	tachesv1 "github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/operateur/api/v1alpha1"
	"github.com/laurentpoirierfr/golang-demystified/examples/approfondissements/operateur/internal/controleur"
)

func main() {
	ctrl.SetLogger(zap.New())
	journal := ctrl.Log.WithName("operateur")

	// Le scheme associe chaque type Go à son groupe, sa version et son type
	// Kubernetes.
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		journal.Error(err, "scheme")
		os.Exit(1)
	}
	if err := tachesv1.AddToScheme(scheme); err != nil {
		journal.Error(err, "scheme")
		os.Exit(1)
	}

	// Le gestionnaire partage un cache des objets surveillés entre les
	// contrôleurs, et gère l'élection d'un chef quand plusieurs copies de
	// l'opérateur tournent en même temps.
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: ":8081",
		LeaderElection:         true,
		LeaderElectionID:       "operateur-taches.taches.example.com",
	})
	if err != nil {
		journal.Error(err, "création du gestionnaire")
		os.Exit(1)
	}

	r := &controleur.ApiTachesReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
	if err := r.SetupWithManager(mgr); err != nil {
		journal.Error(err, "création du contrôleur")
		os.Exit(1)
	}
	mgr.AddHealthzCheck("healthz", healthz.Ping)
	mgr.AddReadyzCheck("readyz", healthz.Ping)

	journal.Info("démarrage de l'opérateur")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		journal.Error(err, "arrêt de l'opérateur")
		os.Exit(1)
	}
}
