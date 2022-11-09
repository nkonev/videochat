CREATE OR REPLACE FUNCTION strip_tags(TEXT) RETURNS TEXT AS $$
SELECT regexp_replace($1, '<[^>]*>', '', 'g')
$$ LANGUAGE SQL;