// Package operateur illustre le chapitre « Kubernetes : écrire un
// opérateur ». Les directives ci-dessous génèrent les méthodes DeepCopy des
// types de l'API, et la définition de la ressource personnalisée (CRD).
package operateur

//go:generate go tool controller-gen object paths=./api/...
//go:generate go tool controller-gen crd paths=./api/... output:crd:artifacts:config=config/crd
//go:generate go tool controller-gen rbac:roleName=operateur-taches paths=./internal/... output:rbac:artifacts:config=config/rbac
