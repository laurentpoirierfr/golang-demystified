CREATE TABLE taches (
    id       INTEGER PRIMARY KEY AUTOINCREMENT,
    titre    TEXT    NOT NULL,
    faite    INTEGER NOT NULL DEFAULT 0,
    creation INTEGER NOT NULL -- millisecondes depuis l'epoch Unix, UTC
);
