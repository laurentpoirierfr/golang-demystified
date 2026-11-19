CREATE TABLE taches (
    id       BIGINT  GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    titre    TEXT    NOT NULL,
    faite    BOOLEAN NOT NULL DEFAULT FALSE,
    creation BIGINT  NOT NULL -- millisecondes depuis l'epoch Unix, UTC
);
