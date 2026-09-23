ALTER TABLE products ALTER COLUMN code SET NOT NULL;
ALTER TABLE products ADD CONSTRAINT products_code_key UNIQUE (code);
