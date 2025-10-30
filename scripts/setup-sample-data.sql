-- ================================================
-- MSSQL Sample Data Setup Script
-- Run this script on your MSSQL Source Database
-- ================================================

-- Create database (if needed)
IF NOT EXISTS (SELECT * FROM sys.databases WHERE name = 'SourceDB')
BEGIN
    CREATE DATABASE SourceDB;
END
GO

USE SourceDB;
GO

-- ================================================
-- Table 1: Users
-- ================================================
IF OBJECT_ID('dbo.Users', 'U') IS NOT NULL
    DROP TABLE dbo.Users;
GO

CREATE TABLE dbo.Users (
    UserID INT PRIMARY KEY IDENTITY(1,1),
    Username NVARCHAR(50) NOT NULL,
    Email NVARCHAR(100) NOT NULL,
    FirstName NVARCHAR(50),
    LastName NVARCHAR(50),
    DateOfBirth DATE,
    CreatedAt DATETIME2 DEFAULT GETDATE(),
    UpdatedAt DATETIME2 DEFAULT GETDATE(),
    IsActive BIT DEFAULT 1,
    Balance DECIMAL(18,2) DEFAULT 0.00
);
GO

-- Insert sample users
INSERT INTO dbo.Users (Username, Email, FirstName, LastName, DateOfBirth, IsActive, Balance) VALUES
('john_doe', 'john.doe@example.com', 'John', 'Doe', '1990-05-15', 1, 1500.00),
('jane_smith', 'jane.smith@example.com', 'Jane', 'Smith', '1985-08-22', 1, 2750.50),
('bob_wilson', 'bob.wilson@example.com', 'Bob', 'Wilson', '1992-03-10', 0, 100.00),
('alice_brown', 'alice.brown@example.com', 'Alice', 'Brown', '1988-11-30', 1, 5000.00),
('charlie_davis', 'charlie.davis@example.com', 'Charlie', 'Davis', '1995-07-18', 1, 875.25);
GO

-- ================================================
-- Table 2: Products
-- ================================================
IF OBJECT_ID('dbo.Products', 'U') IS NOT NULL
    DROP TABLE dbo.Products;
GO

CREATE TABLE dbo.Products (
    ProductID INT PRIMARY KEY IDENTITY(1,1),
    ProductName NVARCHAR(100) NOT NULL,
    Description NVARCHAR(500),
    Price DECIMAL(10,2) NOT NULL,
    CategoryID INT,
    StockQuantity INT DEFAULT 0,
    SKU NVARCHAR(50),
    IsDeleted BIT DEFAULT 0,
    LastModified DATETIME2 DEFAULT GETDATE()
);
GO

-- Insert sample products
INSERT INTO dbo.Products (ProductName, Description, Price, CategoryID, StockQuantity, SKU, IsDeleted) VALUES
('Laptop Pro 15', 'High-performance laptop with 15" display', 1299.99, 1, 50, 'LAP-PRO-15', 0),
('Wireless Mouse', 'Ergonomic wireless mouse', 29.99, 2, 200, 'MOU-WRL-01', 0),
('USB-C Hub', '7-in-1 USB-C Hub with HDMI and card readers', 49.99, 2, 150, 'HUB-USC-07', 0),
('Office Chair', 'Ergonomic office chair with lumbar support', 299.99, 3, 30, 'CHR-OFF-01', 0),
('Standing Desk', 'Electric height-adjustable standing desk', 599.99, 3, 20, 'DSK-STD-01', 0),
('Old Keyboard', 'Discontinued mechanical keyboard', 79.99, 2, 5, 'KEY-MEC-99', 1);
GO

-- ================================================
-- Table 3: Orders
-- ================================================
IF OBJECT_ID('dbo.Orders', 'U') IS NOT NULL
    DROP TABLE dbo.Orders;
GO

CREATE TABLE dbo.Orders (
    OrderID INT PRIMARY KEY IDENTITY(1,1),
    CustomerID INT NOT NULL,
    OrderDate DATETIME2 DEFAULT GETDATE(),
    TotalAmount DECIMAL(10,2) NOT NULL,
    Status NVARCHAR(20) DEFAULT 'Pending',
    ShippingAddress NVARCHAR(200),
    Notes NVARCHAR(500)
);
GO

-- Insert sample orders
INSERT INTO dbo.Orders (CustomerID, OrderDate, TotalAmount, Status, ShippingAddress) VALUES
(1, DATEADD(day, -5, GETDATE()), 1329.98, 'Delivered', '123 Main St, Anytown, USA'),
(2, DATEADD(day, -3, GETDATE()), 349.98, 'Shipped', '456 Oak Ave, Another City, USA'),
(1, DATEADD(day, -1, GETDATE()), 29.99, 'Processing', '123 Main St, Anytown, USA'),
(4, DATEADD(hour, -12, GETDATE()), 599.99, 'Pending', '789 Pine Rd, Some Town, USA'),
(3, DATEADD(day, -30, GETDATE()), 79.99, 'Delivered', '321 Elm St, Old City, USA'),
(2, DATEADD(day, -60, GETDATE()), 1599.98, 'Delivered', '456 Oak Ave, Another City, USA');
GO

-- ================================================
-- Table 4: AuditLog
-- ================================================
IF OBJECT_ID('dbo.AuditLog', 'U') IS NOT NULL
    DROP TABLE dbo.AuditLog;
GO

CREATE TABLE dbo.AuditLog (
    LogID INT PRIMARY KEY IDENTITY(1,1),
    TableName NVARCHAR(50) NOT NULL,
    Action NVARCHAR(20) NOT NULL,
    RecordID INT,
    OldValue NVARCHAR(MAX),
    NewValue NVARCHAR(MAX),
    ModifiedBy NVARCHAR(50),
    ModifiedAt DATETIME2 DEFAULT GETDATE()
);
GO

-- Insert sample audit logs
INSERT INTO dbo.AuditLog (TableName, Action, RecordID, OldValue, NewValue, ModifiedBy) VALUES
('Users', 'INSERT', 1, NULL, 'John Doe created', 'SYSTEM'),
('Users', 'UPDATE', 1, 'Balance: 1000.00', 'Balance: 1500.00', 'admin'),
('Products', 'INSERT', 1, NULL, 'Laptop Pro 15 created', 'SYSTEM'),
('Orders', 'INSERT', 1, NULL, 'Order #1 created', 'SYSTEM'),
('Products', 'UPDATE', 6, 'IsDeleted: 0', 'IsDeleted: 1', 'admin');
GO

-- ================================================
-- Create Views (optional)
-- ================================================
IF OBJECT_ID('dbo.vw_ActiveUsers', 'V') IS NOT NULL
    DROP VIEW dbo.vw_ActiveUsers;
GO

CREATE VIEW dbo.vw_ActiveUsers AS
SELECT 
    UserID,
    Username,
    Email,
    FirstName,
    LastName,
    CreatedAt,
    Balance
FROM dbo.Users
WHERE IsActive = 1;
GO

-- ================================================
-- Print Summary
-- ================================================
PRINT 'Sample data setup completed successfully!';
PRINT '';
PRINT 'Tables created:';
PRINT '- dbo.Users: ' + CAST((SELECT COUNT(*) FROM dbo.Users) AS NVARCHAR(10)) + ' records';
PRINT '- dbo.Products: ' + CAST((SELECT COUNT(*) FROM dbo.Products) AS NVARCHAR(10)) + ' records';
PRINT '- dbo.Orders: ' + CAST((SELECT COUNT(*) FROM dbo.Orders) AS NVARCHAR(10)) + ' records';
PRINT '- dbo.AuditLog: ' + CAST((SELECT COUNT(*) FROM dbo.AuditLog) AS NVARCHAR(10)) + ' records';
PRINT '';
PRINT 'You can now run the sync service with the sample configuration!';
GO
