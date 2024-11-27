# nlp-to-sql

A backend application that enables conversational database interactions, leveraging Retrieval-Augmented Generation (RAG) to generate context-aware, tailored responses. It converts NLP to SQL queries. It takes a textual request and returns a textual response based on the queried data.

#

#### *PROJECT OVERVIEW*

#### Example Question/Request:

- > How many accounts have been opened till date?

#### Generated Query:

```sql
   SELECT COUNT(*) FROM accounts;
```

#### Example Respose:

- > We've got a total of 114 accounts opened so far.

#

#### *SECURITY CONSIDERATIONS*

- Prompts are engineered to ensure that conversations can only lead to **READ** operations:

  - Conditions in place to ensure that queries generated by the AI model are only `SELECT` queries
  - Programmatically, generated queries from the AI model are also checked to ensure that queries  are `SELECT` statements before being triggered.

- Database connection strings provided hashed and then cached for short period of time. But please ensure that temporary connection strings are created before supplying then during usage. Good to note that they are only used programmatically for getting requested data and are not shared.
  - Sensitive data are exempted from the query generated and subsequently from the response provided.
