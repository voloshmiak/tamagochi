-- name: CreatePet :exec
INSERT INTO pets (id, owner_id, name, appearance, satiety, mood, energy, cleanliness, born_at, updated_at, last_actions)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: PetByOwner :one
SELECT * FROM pets WHERE owner_id = $1;

-- name: PetByOwnerForUpdate :one
SELECT * FROM pets WHERE owner_id = $1 FOR UPDATE;

-- name: UpdatePet :execrows
UPDATE pets
SET name = $2, appearance = $3, satiety = $4, mood = $5, energy = $6, cleanliness = $7,
    updated_at = $8, last_actions = $9
WHERE id = $1;

-- name: InsertPetTraits :exec
INSERT INTO pet_traits (pet_id, trait, weight)
SELECT sqlc.arg(pet_id), unnest(sqlc.arg(traits)::text[]), unnest(sqlc.arg(weights)::double precision[]);

-- name: PetTraits :many
SELECT trait, weight FROM pet_traits WHERE pet_id = $1;
