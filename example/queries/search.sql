-- name: SearchUsers :many
SELECT * FROM users
WHERE name = @name
  AND email = @email -- :if @email
  AND phone = @phone -- :if @phone
  AND EXISTS ( -- :if @has_orders
    SELECT 1 FROM orders
    WHERE orders.user_id = users.id
      AND orders.created_at >= @orders_since -- :if @orders_since
  )
ORDER BY id ASC;

-- name: SearchUsersByContact :many
-- Include the combined contact filter only when BOTH email AND phone are provided.
SELECT * FROM users
WHERE name = @name
  AND (email = @email OR phone = @phone) -- :if @email @phone
ORDER BY id ASC;

-- name: SearchUsersOrdered :many
SELECT * FROM users
WHERE name = @name
  AND email = @email -- :if @email
ORDER BY
  created_at DESC, -- :if @order_created_at_desc
  name ASC, -- :if @order_name_asc
  id ASC;
