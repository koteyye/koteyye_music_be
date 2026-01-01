-- Change avatar_url field to avatar_key and increase size
ALTER TABLE users 
RENAME COLUMN avatar_url TO avatar_key;

-- Change the column type to TEXT (unlimited length)
ALTER TABLE users 
ALTER COLUMN avatar_key TYPE TEXT;

-- Update existing avatar URLs to extract keys if needed
-- This handles OAuth avatars that might already be URLs
UPDATE users 
SET avatar_key = CASE 
  WHEN avatar_key LIKE 'http%' THEN NULL  -- Reset external URLs
  ELSE avatar_key  -- Keep existing keys
END;