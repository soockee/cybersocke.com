---
name: "GraphQL Introduction"
slug: "graphql-intro"
tags: ["graphql", "web", "database"]
date: 2021-02-18
description: "An overview of GraphQL, its features, and implementation concepts."
cover: "./graphql.png"
---

GraphQL is an open-source query language and data manipulation framework. Developed by Facebook in 2012, it was released to the public in 2015 and has been managed by the GraphQL Foundation since 2018. As a specification, GraphQL has been implemented across various platforms.

The core philosophy of GraphQL is encapsulated in the motto: *"Get many resources in a single request."* By consolidating multiple REST endpoints into a single endpoint, GraphQL allows for complex queries to fetch data efficiently. Additionally, it provides strong typing through schema definitions.

## CRUD Operations

### Create
```graphql
mutation test {
    createUser (props: {
        username: "JonnyD",
        password: "4204242",
        firstName: "John",
        lastName: "Doe"
    }) {
        id
    }
}
```

### Read
```graphql
query users {
   users { id, firstName, lastName }
}
```

### Update
```graphql
mutation updUser {
    updateUser (
        id: 5,
        props: {
            username: "JonnyD",
            password: "newPwd",
            firstName: "John",
            lastName: "Doe"
        }) {
        id
    }
}
```

### Delete
```graphql
mutation delUser {
    deleteUser (id: 5) {
        id
    }
}
```

## Comparable Technologies

- 1980: RPC
- 1998: XML-RPC
- 1999: SOAP
- 2000: REST
- 2007: Thrift
- 2012: GraphQL
- 2015: gRPC

## GraphQL Backend Concept

GraphQL operates over HTTP, delivering data via a single endpoint in JSON format (or serialized forms). Users specify the data they require through queries, enabling precise access to the desired object properties or values. To achieve this, object schemas, queries, and mutations must be defined. GraphQL supports basic types such as `String` and `Float`.

In addition, *Data Access Objects* (DAOs) must be defined to manage communication between the application and the database.

### Example: User Implementation in GraphQL

```javascript
import {
    GraphQLFieldConfigMap,
    GraphQLID,
    GraphQLInputObjectType,
    GraphQLList,
    GraphQLNonNull,
    GraphQLObjectType,
    GraphQLString
} from 'graphql';
import {GenderType} from './GenderType';
import {Pet} from './PetSchema';
import {UserDAO} from '../neo4j/UserDAO';
import * as crypto from "crypto-js";

export const User = new GraphQLObjectType({
    name: 'User',
    fields: {
        id: { type: GraphQLID, description: 'User identifier - keycloak Id.' },
        username: { type: GraphQLNonNull(GraphQLString), description: 'The name of the user.' },
        password: { type: GraphQLString, description: 'The hashed password of the user.' },
        gender: { type: GenderType, description: 'The gender of the user.' },
        firstName: { type: GraphQLString, description: 'The first name of the user.' },
        lastName: { type: GraphQLString, description: 'The last name of the user.' },
        pets: { type: new GraphQLList(Pet), description: 'The list of pets the user has.' }
    }
});

export const UserInput = new GraphQLInputObjectType({
    name: 'UserInput',
    fields: {
        username: { type: GraphQLString },
        password: { type: GraphQLString },
        gender: { type: GenderType },
        firstName: { type: GraphQLString },
        lastName: { type: GraphQLString }
    }
});

export const UserQueries = {
    user: {
        type: User,
        args: { id: { type: GraphQLID } },
        resolve: async (root, args) => {
            const { id } = args;
            return await UserDAO.get(id);
        }
    },
    users: {
        type: new GraphQLList(User),
        resolve: async () => {
            return await UserDAO.getAll();
        }
    }
};

export const UserMutation = {
    createUser: {
        type: User,
        args: { props: { type: UserInput } },
        resolve: async (root, args) => {
            const { props } = args;

            if (props?.password) {
                props.password = crypto.SHA256(props.password).toString();
            }

            return await UserDAO.create(props);
        }
    },
    updateUser: {
        type: User,
        args: {
            id: { type: GraphQLID },
            props: { type: UserInput }
        },
        resolve: async (root, args) => {
            const { id, props } = args;

            if (props?.password) {
                props.password = crypto.SHA256(props.password).toString();
            }

            return await UserDAO.update(id, props);
        }
    },
    deleteUser: {
        type: User,
        args: { id: { type: GraphQLID } },
        resolve: async (root, args) => {
            const { id } = args;
            await UserDAO.delete(id);
        }
    }
};
```

## GraphQL Frontend Concept

There are numerous client libraries for GraphQL. The best library depends on the use case. Popular libraries include Apollo, urql, Micro GraphQL React, and Grafoo.

## GraphQL Persistence Concept

GraphQL is solely a query language. Data persistence is managed using external databases or filesystems, such as SQL databases (e.g., MySQL) or NoSQL databases (e.g., MongoDB). Graph databases like Neo4j are particularly well-suited due to their performance.

### Example: Neo4j Database Connection

```javascript
import neo4j, { Driver } from 'neo4j-driver';

export class Neo4JDriver {
    public static instance: Driver;

    public static createDatabaseConnection(url: string, username: string, password: string): Driver {
        this.instance = neo4j.driver(url, neo4j.auth.basic(username, password));
        return this.instance;
    }
}
```

### Configuration Example

```json
{
  "SERVER_PORT": "8000",
  "SESSION": {
    "SECRET": "keyboard mouse",
    "MAX_AGE": 60000,
    "RESAVE": true,
    "SAVE_UNINITIALIZED": true
  },
  "NEO4J": {
    "URL": "neo4j://localhost",
    "USERNAME": "neo4j",
    "PASSWORD": "password"
  },
  "CRYPTO": {
    "KEY": "applepie"
  }
}
```

```typescript
import express, { Express } from 'express';
import session, { SessionOptions } from 'express-session';
import * as CONFIG from './start.cfg.json';
import { graphqlHTTP } from 'express-graphql';
import { schema } from './schemes/_schema';
import { Neo4JDriver } from './utils/Neo4JDriver';

const application: Express = express();

const sessionOptions: SessionOptions = {
    secret: CONFIG.SESSION.SECRET,
    cookie: { maxAge: CONFIG.SESSION.MAX_AGE },
    resave: CONFIG.SESSION.RESAVE,
    saveUninitialized: CONFIG.SESSION.SAVE_UNINITIALIZED
};

Neo4JDriver.createDatabaseConnection(CONFIG.NEO4J.URL, CONFIG.NEO4J.USERNAME, CONFIG.NEO4J.PASSWORD);

application.use(session(sessionOptions));

application.use('/graphql', graphqlHTTP({
    schema,
    pretty: true,
    graphiql: true
}));

application.listen(CONFIG.SERVER_PORT, () => {
    console.info(`GraphQL Server started on Port ${CONFIG.SERVER_PORT}.`);
});
```

### Source

This content is based on a seminar conducted as part of the Web Technology module at THM.

Presenters:
- Sebastian Enns
- Tymoteusz Mucha
- Felix Müscher

