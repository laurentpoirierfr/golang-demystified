ALTER TABLE livres
    ADD COLUMN exemplaires INTEGER NOT NULL DEFAULT 1 CHECK (exemplaires >= 0);
