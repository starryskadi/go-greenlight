CREATE UNIQUE INDEX IF NOT EXISTS movies_genres_movie_id_genre_id_idx
ON movies_genres (movie_id, genre_id);