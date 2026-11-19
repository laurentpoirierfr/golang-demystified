package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

	// Base est la chaîne de connexion à la base (TACHES_BASE). Par défaut,
	// une base SQLite dans le conteneur.
	// +optional
	Base string `json:"base,omitempty"`
}

// ApiTachesStatus décrit l'état observé : ce que l'opérateur constate.
type ApiTachesStatus struct {
	// ReplicasPretes est le nombre d'instances prêtes à répondre.
	ReplicasPretes int32 `json:"replicasPretes,omitempty"`

	// Conditions suit les conventions de Kubernetes : la condition « Pret »
	// indique si toutes les instances demandées sont prêtes.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ApiTaches est une ressource personnalisée : une instance de l'API des
// tâches, que l'opérateur déploie et maintient dans la grappe.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Replicas",type=integer,JSONPath=`.spec.replicas`
// +kubebuilder:printcolumn:name="Prêtes",type=integer,JSONPath=`.status.replicasPretes`
// +kubebuilder:printcolumn:name="Âge",type=date,JSONPath=`.metadata.creationTimestamp`
type ApiTaches struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApiTachesSpec   `json:"spec"`
	Status ApiTachesStatus `json:"status,omitempty"`
}

// ApiTachesList est une liste d'ApiTaches.
//
// +kubebuilder:object:root=true
type ApiTachesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApiTaches `json:"items"`
}
