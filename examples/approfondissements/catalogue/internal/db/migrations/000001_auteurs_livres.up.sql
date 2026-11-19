CREATE TABLE auteurs (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    nom  TEXT   NOT NULL UNIQUE
);

CREATE TABLE livres (
    id        BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    isbn      TEXT        NOT NULL UNIQUE,
    titre     TEXT        NOT NULL,
    auteur_id BIGINT      NOT NULL REFERENCES auteurs (id),
    annee     INTEGER,    -- facultative : NULL si inconnue
    ajout     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX livres_auteur_idx ON livres (auteur_id);
