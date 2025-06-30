psql -h localhost -U mars -d mars -c "\COPY ( $(cat export.sql) ) TO '~/Desktop/output.csv' WITH CSV HEADER;"
