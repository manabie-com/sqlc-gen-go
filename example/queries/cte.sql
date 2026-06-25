-- name: GetUsersWithOrderTotals :many
WITH order_totals AS (
    SELECT user_id, COUNT(*) as order_count, COALESCE(SUM(amount), 0) as total_spent
    FROM orders
    GROUP BY user_id
)
SELECT u.id, u.name, u.email, ot.order_count, ot.total_spent
FROM users u
INNER JOIN order_totals ot ON ot.user_id = u.id
ORDER BY ot.total_spent DESC;

-- name: GetTopSpenders :many
WITH order_totals AS (
    SELECT user_id, COUNT(*) as order_count, COALESCE(SUM(amount), 0) as total_spent
    FROM orders
    GROUP BY user_id
    HAVING SUM(amount) >= @min_spent
),
user_info AS (
    SELECT u.id, u.name, u.email, ot.order_count, ot.total_spent
    FROM users u
    INNER JOIN order_totals ot ON ot.user_id = u.id
)
SELECT * FROM user_info
ORDER BY total_spent DESC
LIMIT @max_results;

-- name: GetUserRank :one
WITH ranked_users AS (
    SELECT u.id as user_id, u.name, u.email,
           COALESCE(SUM(o.amount), 0) as total_spent,
           RANK() OVER (ORDER BY COALESCE(SUM(o.amount), 0) DESC) as spend_rank
    FROM users u
    LEFT JOIN orders o ON o.user_id = u.id
    GROUP BY u.id, u.name, u.email
)
SELECT user_id, name, email, total_spent, spend_rank FROM ranked_users
WHERE user_id = @user_id;

-- name: UpsertAndReturnUser :one
WITH upserted AS (
    INSERT INTO users (name, email)
    VALUES (@name, @email)
    ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name
    RETURNING *
)
SELECT * FROM upserted;

-- name: DeleteUserAndReturnOrders :many
WITH deleted_user AS (
    DELETE FROM orders WHERE user_id = @user_id RETURNING *
)
SELECT * FROM deleted_user;

-- name: GetUserSpendingTiers :many
-- Multi-step pipeline: aggregate → percentile → classify into tiers
WITH order_stats AS (
    SELECT user_id,
           COUNT(*) as order_count,
           COALESCE(SUM(amount), 0) as total_spent,
           COALESCE(AVG(amount), 0) as avg_order,
           COALESCE(MAX(amount), 0) as max_order,
           MIN(o.created_at) as first_order_at,
           MAX(o.created_at) as last_order_at
    FROM orders o
    GROUP BY user_id
),
spending_percentiles AS (
    SELECT user_id, order_count, total_spent, avg_order, max_order,
           first_order_at, last_order_at,
           PERCENT_RANK() OVER (ORDER BY total_spent) as spend_percentile,
           NTILE(4) OVER (ORDER BY total_spent) as spend_quartile
    FROM order_stats
),
tiered_users AS (
    SELECT sp.*,
           CASE
               WHEN spend_quartile = 4 THEN 'platinum'
               WHEN spend_quartile = 3 THEN 'gold'
               WHEN spend_quartile = 2 THEN 'silver'
               ELSE 'bronze'
           END as tier
    FROM spending_percentiles sp
)
SELECT u.id, u.name, u.email,
       tu.order_count, tu.total_spent, tu.avg_order, tu.max_order,
       tu.first_order_at, tu.last_order_at,
       tu.spend_percentile, tu.tier
FROM users u
INNER JOIN tiered_users tu ON tu.user_id = u.id
ORDER BY tu.total_spent DESC;

-- name: GetRevenueGrowth :many
-- Generate a date series and compute month-over-month revenue with growth rate
WITH monthly_revenue AS (
    SELECT DATE_TRUNC('month', created_at)::DATE as month,
           COUNT(*) as order_count,
           SUM(amount) as revenue
    FROM orders
    WHERE created_at >= @since AND created_at < @until
    GROUP BY DATE_TRUNC('month', created_at)
),
with_lag AS (
    SELECT month, order_count, revenue,
           LAG(revenue) OVER (ORDER BY month) as prev_revenue,
           LAG(order_count) OVER (ORDER BY month) as prev_order_count
    FROM monthly_revenue
),
with_growth AS (
    SELECT month, order_count, revenue,
           prev_revenue,
           CASE
               WHEN prev_revenue IS NOT NULL AND prev_revenue > 0
               THEN ROUND(((revenue - prev_revenue) / prev_revenue) * 100, 2)
               ELSE NULL
           END as growth_pct,
           SUM(revenue) OVER (ORDER BY month) as cumulative_revenue
    FROM with_lag
)
SELECT month, order_count, revenue, prev_revenue, growth_pct, cumulative_revenue
FROM with_growth
ORDER BY month;

-- name: GetUserCohortRetention :many
-- Cohort analysis: users grouped by signup month with ordering activity counts
WITH user_cohorts AS (
    SELECT users.id as user_id,
           DATE_TRUNC('month', users.created_at)::DATE as cohort_month
    FROM users
),
cohort_orders AS (
    SELECT user_cohorts.cohort_month,
           COUNT(DISTINCT user_cohorts.user_id) as active_users,
           COUNT(orders.id) as total_orders
    FROM user_cohorts
    INNER JOIN orders ON orders.user_id = user_cohorts.user_id
    WHERE orders.created_at >= user_cohorts.cohort_month
      AND orders.created_at < user_cohorts.cohort_month + INTERVAL '1 year'
    GROUP BY user_cohorts.cohort_month
)
SELECT * FROM cohort_orders
ORDER BY cohort_month;

-- name: TransferUserOrders :many
-- Writable CTE chain: update orders to new user, then return the transferred orders with both user names
WITH transferred AS (
    UPDATE orders
    SET user_id = @to_user_id
    WHERE user_id = @from_user_id AND status = @status
    RETURNING *
),
from_user AS (
    SELECT name as from_name FROM users WHERE id = @from_user_id
),
to_user AS (
    SELECT name as to_name FROM users WHERE id = @to_user_id
)
SELECT t.id, t.amount, t.status, t.created_at,
       fu.from_name, tu.to_name
FROM transferred t
CROSS JOIN from_user fu
CROSS JOIN to_user tu
ORDER BY t.created_at;

-- name: SearchUsersWithStats :many
-- CTE combined with dynamic filters on base table columns
WITH user_order_stats AS (
    SELECT orders.user_id,
           COUNT(*) as order_count,
           COALESCE(SUM(orders.amount), 0) as total_spent,
           MAX(orders.created_at) as last_order_at
    FROM orders
    GROUP BY orders.user_id
)
SELECT users.id, users.name, users.email, users.created_at,
       COALESCE(user_order_stats.order_count, 0) as order_count,
       COALESCE(user_order_stats.total_spent, 0) as total_spent,
       user_order_stats.last_order_at
FROM users
LEFT JOIN user_order_stats ON user_order_stats.user_id = users.id
WHERE 1 = 1
  AND users.name = @name -- :if @name
  AND users.email = @email -- :if @email
ORDER BY
  users.created_at DESC, -- :if @order_by_created
  users.name ASC;

-- name: GetActiveUsersWithSubquery :many
-- CTE with nested subquery in SELECT
WITH user_activity AS (
    SELECT u.id as user_id, u.name,
           (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) as order_count,
           (SELECT COALESCE(SUM(o.amount), 0) FROM orders o WHERE o.user_id = u.id) as total_spent
    FROM users u
)
SELECT user_activity.user_id, user_activity.name,
       user_activity.order_count, user_activity.total_spent
FROM user_activity
WHERE user_activity.order_count > 0
ORDER BY user_activity.total_spent DESC;

-- name: GetAllEntities :many
-- CTE with UNION ALL
WITH all_entities AS (
    SELECT id as entity_id, name as entity_name, 'user' as entity_type, created_at
    FROM users
    UNION ALL
    SELECT id as entity_id, status as entity_name, 'order' as entity_type, created_at
    FROM orders
)
SELECT all_entities.entity_id, all_entities.entity_name,
       all_entities.entity_type, all_entities.created_at
FROM all_entities
ORDER BY all_entities.created_at DESC;

-- name: GetUserTiersBySubquery :many
-- CTE with CASE containing subquery
WITH user_tiers AS (
    SELECT u.id as user_id, u.name,
           CASE
               WHEN (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) > 10 THEN 'platinum'
               WHEN (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) > 5 THEN 'gold'
               WHEN (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) > 0 THEN 'silver'
               ELSE 'bronze'
           END as tier
    FROM users u
)
SELECT user_tiers.user_id, user_tiers.name, user_tiers.tier
FROM user_tiers
ORDER BY user_tiers.user_id;

-- name: GetUsersWithOrders :many
-- CTE with EXISTS filter
WITH active_users AS (
    SELECT id, name, email
    FROM users u
    WHERE EXISTS (SELECT 1 FROM orders WHERE user_id = u.id ORDER BY created_at DESC LIMIT 1)
)
SELECT active_users.id, active_users.name, active_users.email
FROM active_users
ORDER BY active_users.name;

-- name: GetExclusiveHighValueUsers :many
-- CTE with EXCEPT
WITH high_value_users AS (
    SELECT DISTINCT user_id FROM orders WHERE amount > 100
    EXCEPT
    SELECT DISTINCT user_id FROM orders WHERE status = 'cancelled'
)
SELECT u.id, u.name, u.email
FROM users u
INNER JOIN high_value_users hvu ON hvu.user_id = u.id
ORDER BY u.name;
