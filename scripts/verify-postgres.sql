-- ================================================
-- PostgreSQL Verification Script
-- Run this script on your PostgreSQL Target Database
-- to verify the synced data
-- ================================================

\c targetdb

-- Check if tables exist
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY tablename;

-- Verify Users table
\echo '\n=== Users Table ==='
SELECT 
    COUNT(*) as total_users,
    SUM(CASE WHEN "IsActive" = true THEN 1 ELSE 0 END) as active_users,
    SUM("Balance") as total_balance
FROM public.users;

SELECT * FROM public.users LIMIT 5;

-- Verify Products table
\echo '\n=== Products Table ==='
SELECT 
    COUNT(*) as total_products,
    SUM(CASE WHEN "IsDeleted" = false THEN 1 ELSE 0 END) as active_products,
    AVG("Price") as average_price
FROM public.products;

SELECT "ProductName", "Price", "StockQuantity" FROM public.products LIMIT 5;

-- Verify Orders table
\echo '\n=== Orders Table ==='
SELECT 
    COUNT(*) as total_orders,
    SUM("TotalAmount") as total_revenue,
    COUNT(DISTINCT "CustomerID") as unique_customers
FROM public.orders;

SELECT "OrderID", "CustomerID", "OrderDate", "TotalAmount", "Status" FROM public.orders LIMIT 5;

-- Verify AuditLog table
\echo '\n=== Audit Log Table ==='
SELECT 
    COUNT(*) as total_logs,
    COUNT(DISTINCT "TableName") as tables_audited
FROM public.audit_log;

SELECT "TableName", "Action", "ModifiedBy", "ModifiedAt" FROM public.audit_log LIMIT 5;

\echo '\nVerification complete!'
