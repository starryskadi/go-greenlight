CREATE TABLE IF NOT EXISTS movies (
    id bigserial PRIMARY KEY, 
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    title text NOT NULL,
    year integer NOT NULL,
    runtime integer NOT NULL,
    version integer NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS genres (
    id bigserial PRIMARY KEY, 
    title text NOT NULL
);

ALTER TABLE genres
ADD CONSTRAINT title_unique UNIQUE (title);

CREATE TABLE IF NOT EXISTS movies_genres (
    movie_id int,
    genre_id int
);

ALTER TABLE movies_genres
ADD CONSTRAINT pk_movies_genres PRIMARY KEY (movie_id, genre_id);

ALTER TABLE movies_genres
ADD CONSTRAINT fk_movie_id FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE;

ALTER TABLE movies_genres
ADD CONSTRAINT fk_genre_id FOREIGN KEY (genre_id) REFERENCES genres(id) ON DELETE CASCADE;