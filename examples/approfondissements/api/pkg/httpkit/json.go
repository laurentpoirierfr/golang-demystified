package httpkit

import (
	"encoding/json"
	"io"
	"net/http"
)

// ErrRequete signale une requête mal formée, à traduire en statut 400.
type ErrRequete struct{ Raison string }

func (e *ErrRequete) Error() string { return e.Raison }

// TailleMax est la taille maximale d'un corps JSON accepté par LitJSON.
const TailleMax = 1 << 20

// LitJSON décode un unique objet JSON, de 1 Mo au plus, sans champ inconnu.
func LitJSON(w http.ResponseWriter, r *http.Request, dest any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, TailleMax))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return &ErrRequete{"JSON invalide : " + err.Error()}
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return &ErrRequete{"un seul objet JSON est attendu"}
	}
	return nil
}

// EcritJSON écrit v en JSON avec le statut donné.
func EcritJSON(w http.ResponseWriter, statut int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statut)
	json.NewEncoder(w).Encode(v)
}
