-- Timepris på leverandører/håndværkere, til overslag og sammenligning.
ALTER TABLE suppliers ADD COLUMN hourly_rate numeric(12,2);
