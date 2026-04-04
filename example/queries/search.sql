-- name: SearchUsers :many
SELECT * FROM users
WHERE name = @name
  AND email = @email -- :if @email
  AND phone = @phone -- :if @phone
ORDER BY id ASC;
