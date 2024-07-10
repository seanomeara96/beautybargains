CREATE TABLE IF NOT EXISTS brands (
    id INTEGER PRIMARY KEY,
    name TEXT,
    path TEXT
);

CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY,
    website_id INTEGER,
    description TEXT,
    src_url TEXT,
    link TEXT,
    timestamp TIMESTAMP,
    author_id INTEGER,
    FOREIGN KEY (website_id) REFERENCES websites(website_id)
);

CREATE TABLE IF NOT EXISTS hashtags (
    id INTEGER PRIMARY KEY,
    phrase TEXT
);

CREATE TABLE IF NOT EXISTS post_hashtags (
    id INTEGER PRIMARY KEY,
    post_id INTEGER,
    hashtag_id INTEGER,
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (hashtag_id) REFERENCES hashtags(id)
);


