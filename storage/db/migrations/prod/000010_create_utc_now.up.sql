CREATE FUNCTION utc_now() returns TIMESTAMP AS $$ SELECT now() at time zone 'utc' $$ LANGUAGE SQL;
