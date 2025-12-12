# Guidance for Python Scripts

- Use polar.rs instead of pandas. Check all methods to ensure they are valid polars methods, some pandas methods are named differently than polars.
- Don't add complexity with factory patterns to prepare for different databases. These scripts will only interact with postgres.
- Polars has a function called read_database_uri, but it never works in this environment. Run the query using psycopg and load the query results using a pl.dataframe function
- For python code you generate, format as PEP8.
