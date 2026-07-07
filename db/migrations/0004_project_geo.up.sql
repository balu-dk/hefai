-- Projektets geoposition (fra adresseopslag). Bruges til luftfoto,
-- lokalplan-opslag og som standard-forankring for tegninger. UTM-koordinater
-- (ETRS89/UTM32) gemmes sammen med lat/lon, fordi Plandata-opslag kræver dem.
ALTER TABLE projects ADD COLUMN latitude  double precision;
ALTER TABLE projects ADD COLUMN longitude double precision;
ALTER TABLE projects ADD COLUMN utm_x     double precision;
ALTER TABLE projects ADD COLUMN utm_y     double precision;
