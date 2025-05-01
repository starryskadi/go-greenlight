ALTER TABLE movies ADD CONSTRAINT movies_runtime_check CHECK (runtime >= 0); 

ALTER TABLE movies ADD CONSTRAINT movies_year_check CHECK (year BETWEEN 1888 AND date_part('year', NOW()))