CREATE TABLE IF NOT EXISTS items (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  category VARCHAR(50) NOT NULL,
  material VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  note TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  INDEX idx_category_material (category, material),
  INDEX idx_material (material)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
