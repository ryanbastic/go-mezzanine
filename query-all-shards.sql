DO $$
  DECLARE
      tbl  RECORD;
      row  RECORD;
      cnt  INT;
  BEGIN
      FOR tbl IN
          SELECT table_schema, table_name
          FROM information_schema.tables
          WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
            AND table_type = 'BASE TABLE'
          ORDER BY table_name
      LOOP
          cnt := 0;
          FOR row IN
              EXECUTE format('SELECT * FROM %I.%I', tbl.table_schema, tbl.table_name)
          LOOP
              IF cnt = 0 THEN
                  RAISE NOTICE '--- %.% ---', tbl.table_schema, tbl.table_name;
              END IF;
              cnt := cnt + 1;
              RAISE NOTICE '%', row;
          END LOOP;
          IF cnt > 0 THEN
              RAISE NOTICE '(%s rows)', cnt;
              RAISE NOTICE '';
          END IF;
      END LOOP;
END $$;
