// Package v1alpha1 définit la ressource personnalisée ApiTaches, du groupe
// taches.example.com, en version v1alpha1.
//
// Un package d'API ne dépend que de k8s.io/apimachinery : d'autres projets
// (clients, outils) peuvent l'importer sans tirer controller-runtime.
//
// +kubebuilder:object:generate=true
// +groupName=taches.example.com
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion identifie le groupe et la version de l'API.
	GroupVersion = schema.GroupVersion{Group: "taches.example.com", Version: "v1alpha1"}

	// SchemeBuilder enregistre les types du groupe dans un runtime.Scheme.
	SchemeBuilder = runtime.NewSchemeBuilder(ajouteTypes)

	// AddToScheme ajoute les types du groupe à un runtime.Scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func ajouteTypes(s *runtime.Scheme) error {
	s.AddKnownTypes(GroupVersion, &ApiTaches{}, &ApiTachesList{})
	metav1.AddToGroupVersion(s, GroupVersion)
	return nil
}
