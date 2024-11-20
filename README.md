# nlp-to-sql
A backend application that converts NLP to SQL queries.


### SECURITY CONSIDERATIONS
- Prompts are engineered to ensure that conversations can only lead to #Read# operations:
-- Conditions in place to ensure that queries generated by the AI model are only `SELECT` queries
-- Programmatically, generated queries from the AI model are also checked to ensure that queries are `SELECT` statements before being triggered.
-- Database connection strings provided hashed and then cached for short period of time. But please ensure that temporary connection strings are created before supplying then during usage. Good to note that they are only used programmatically for getting requested data and are not shared.
-- Certain measures have been put in place to