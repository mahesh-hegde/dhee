This is a statically rendered website.

Routes have to be controlled and validated using golang echo server framework.

Controller methods on `DheeController` need to be created for the following routes. Each of them should validate the parameters and call the appropriate service method.

For each controller method (unless a JSON-output is indicated), a stub template (with no content but taking a .data parameter and rendering it as code block for inspection) should be created in template/ folder and called from the controller endpoint. We will take up the frontend later.

## get excerpt (s)

```
/scriptures/{scriptureName}/excerpts/path
```

Since path is a slice of numbers internally, we represent it in the URL path by separating the numbers by a dot. Eg: 

```
GET /scriptures/rigveda/excerpts/4.1.1
```

The last element of this can be used to represent a range in the smallest unit. Eg

```
GET /scriptures/rigveda/excerpts/4.1.1-4
```

This param should be passed to the service method as [[4,1,1], [4,1,2], [4,1,3], [4,1,4]] (i.e range inclusive).

## Search
Search CAN be across multiple scriptures, so all fields of scripture.SearchParams (Don't confuse this with dictionary.SearchParams) are represented by URL query parameters, and there's no path param

```
GET /scripture-search/?params?query=<query>&tl=slp1&mode=exact
```

Query is mandatory, whereas tl will default to SLP1 and mode will default to exact. a list of scriptures can be taken as 

## Dictionaries: Get
```
GET /dictionaries/{dictionaryName}/words/{slp1Word}
```

Note that one word can have multiple matches.

## Dictionaries: Search

```
GET /dictionaries/{dictionaryName}/search?q={query}&tl=slp1&mode=prefix
```
Exactly one of `query` or `textQuery` should be provided.

## Dictionaries: suggestions

```
GET /dictionaries/{dictionaryName}/suggestions?q=query&tl=slp1
```

This is one endpoint which should actually return JSON.

# Server setup and error handling

Setup a golang `echo` (already installed) server in echo_server.go, by creating a method

```
StartServer(controller DheeController, config DheeConfig, host int, port int)
```

It should setup and start the server on specified bind host and port. `StartServer` call should be the only addition to `main.go`.

finally, do not add additional dependencies without asking. We are very strict about supply chain
