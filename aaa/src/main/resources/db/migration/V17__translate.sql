-- https://redbyte.eu/en/blog/transliteration-in-postgresql/
CREATE OR REPLACE FUNCTION cyrillic_transliterate(p_string text) RETURNS character varying AS
$BODY$
SELECT replace(replace(replace(replace(replace(replace(replace(replace(translate(lower($1),'абвгдеёзийклмнопрстуфхцэы','abvgdeezijklmnoprstufхcey'), 'ж', 'zh'), 'ч', 'ch'), 'ш', 'sh'), 'щ', 'sch'), 'ъ', ''), 'ю', 'yu'), 'я', 'ya'), 'ь', '');
$BODY$
LANGUAGE SQL IMMUTABLE COST 100;
