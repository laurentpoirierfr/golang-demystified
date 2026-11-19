-- name: AssureAuteur :one
-- Crée l'auteur, ou renvoie celui qui porte déjà ce nom.
INSERT INTO auteurs (nom) VALUES ($1)
ON CONFLICT (nom) DO UPDATE SET nom = EXCLUDED.nom
RETURNING *;

-- name: CreeLivre :one
INSERT INTO livres (isbn, titre, auteur_id, annee, exemplaires)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: Livre :one
SELECT * FROM livres WHERE isbn = $1;

-- name: LivresAvecAuteur :many
SELECT sqlc.embed(livres), auteurs.nom AS auteur
FROM livres
JOIN auteurs ON auteurs.id = livres.auteur_id
ORDER BY auteurs.nom, livres.titre;

-- name: RechercheLivres :many
-- Recherche par titre, et éventuellement par année minimale.
SELECT * FROM livres
WHERE titre ILIKE '%' || sqlc.arg(texte)::text || '%'
  AND (sqlc.narg(annee_min)::integer IS NULL OR annee >= sqlc.narg(annee_min))
ORDER BY titre
LIMIT sqlc.arg(limite);

-- name: StatistiquesParAuteur :many
SELECT auteurs.nom, COUNT(livres.id) AS livres, COALESCE(SUM(livres.exemplaires), 0)::bigint AS exemplaires
FROM auteurs
LEFT JOIN livres ON livres.auteur_id = auteurs.id
GROUP BY auteurs.nom
ORDER BY auteurs.nom;

-- name: PrendExemplaire :execrows
-- Décrémente le stock, seulement s'il reste un exemplaire.
UPDATE livres SET exemplaires = exemplaires - 1
WHERE isbn = $1 AND exemplaires > 0;

-- name: SupprimeLivre :exec
DELETE FROM livres WHERE isbn = $1;
