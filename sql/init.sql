-- Create the Products Table with URL and LastCrawled columns
CREATE TABLE Products (
    ProductID INTEGER PRIMARY KEY,
    WebsiteID INTEGER,
    ProductName TEXT,
    Description TEXT,
    Category TEXT,
    URL TEXT, -- New column for the product's URL
    FOREIGN KEY (WebsiteID) REFERENCES Websites(WebsiteID)
);

-- Create the Websites Table
CREATE TABLE Websites (
    WebsiteID INTEGER PRIMARY KEY,
    WebsiteName TEXT NOT NULL,
    URL TEXT NOT NULL,
    Country TEXT
);

-- Create the Price Data Table
CREATE TABLE PriceData (
    PriceID INTEGER PRIMARY KEY,
    ProductID INTEGER,
    Price REAL,
    Currency TEXT,
    Timestamp DATETIME,
    FOREIGN KEY (ProductID) REFERENCES Products(ProductID)
);

