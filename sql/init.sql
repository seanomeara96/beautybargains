CREATE TABLE brands (
    id INTEGER PRIMARY KEY,
    name TEXT,
    path TEXT,
    score float64 default 0
);

CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    website_id INTEGER,
    description TEXT,
    src_url TEXT,
    link TEXT,
    timestamp TIMESTAMP,
    author_id INTEGER,
    score FLOAT64 DEFAULT 0,
    FOREIGN KEY (website_id) REFERENCES websites(website_id)
);

CREATE TABLE hashtags (id INTEGER PRIMARY KEY, phrase TEXT);

CREATE TABLE post_hashtags (
    id INTEGER PRIMARY KEY,
    post_id INTEGER,
    hashtag_id INTEGER,
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (hashtag_id) REFERENCES hashtags(id)
);

CREATE TABLE subscribers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    full_name TEXT,
    consent BOOLEAN NOT NULL CHECK (consent IN (0, 1)),
    signup_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    verification_token TEXT UNIQUE,
    is_verified BOOLEAN DEFAULT 0,
    preferences TEXT
);

CREATE TABLE sqlite_sequence(name, seq);

CREATE TABLE categories(
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    parent_id INTEGER DEFAULT 0,
    name TEXT NOT NULL,
    url TEXT NOT NULL
);

CREATE TABLE post_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER,
    category_id INTEGER,
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);

CREATE TABLE post_brands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id INTEGER,
    brand_id INTEGER,
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (brand_id) REFERENCES brands(id)
);

CREATE TABLE coupon_codes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL,
    description TEXT NOT NULL,
    valid_until DATETIME,
    first_seen DATETIME,
    website_id INTEGER
);