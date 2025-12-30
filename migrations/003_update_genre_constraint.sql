-- Update genre constraint to support expanded list with lowercase keys
-- Migration: Update albums genre check constraint

-- Drop the old constraint
ALTER TABLE albums DROP CONSTRAINT albums_genre_check;

-- Add new constraint with expanded genre list in lowercase
ALTER TABLE albums ADD CONSTRAINT albums_genre_check 
    CHECK (genre IN (
        'pop', 'rock', 'hip-hop', 'rap', 'indie', 'electronic', 'house', 'techno',
        'jazz', 'blues', 'classical', 'metal', 'punk', 'r-n-b', 'soul', 'folk',
        'reggae', 'country', 'latin', 'k-pop', 'soundtrack', 'lo-fi', 'chanson'
    ));

-- Update existing album genres to lowercase (if any exist)
UPDATE albums SET genre = LOWER(genre) WHERE genre IS NOT NULL;

-- Update comment for documentation
COMMENT ON COLUMN albums.genre IS 'Music genre from expanded list (lowercase): pop, rock, hip-hop, rap, indie, electronic, house, techno, jazz, blues, classical, metal, punk, r-n-b, soul, folk, reggae, country, latin, k-pop, soundtrack, lo-fi, chanson';