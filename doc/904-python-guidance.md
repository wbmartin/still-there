# Guidance for Python Scripts

- Today the only Python here is the pytest suite in `tests/`: pytest, requests, pytest-html and pyotp (`tests/requirements.txt`), plus the stdlib. Nothing in `bin/` or `scripts/` is Python. The polars and psycopg notes below apply only when a script needs dataframes or a direct database connection; a short script does not pull them in.
- Use polar.rs instead of pandas. Check all methods to ensure they are valid polars methods, some pandas methods are named differently than polars.
- Don't add complexity with factory patterns to prepare for different databases. These scripts will only interact with postgres.
- Polars has a function called read_database_uri, but it never works in this environment. Run the query using psycopg and load the query results using a pl.dataframe function
- For python code you generate, format as PEP8.
