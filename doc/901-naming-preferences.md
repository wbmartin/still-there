# Code Naming Preferences

- when you name a list variable, do not use the plural form. instead suffix it with \_list or List (depending on the appropriate naming convention in use)
- prioritize readability when naming a variable - choose more verbose names like "tagCache" instead of "tc"
- do not shorten variables that implement a class name e.g. a variable implementing the ExampleRepository should be exampleRepository; not exampleRepo, not exRepo.  One exception is the Golang method receiver - it can be shortened to 1 char.
- For function names (golang, & TS), use camelCase with descriptive names, e.g., getUserList instead of getUsers because it returns a list. Making it plural is easy to confuse with a single object.
- For API endpoints, use singular nouns with List suffix for collections, e.g., /exampleList instead of /examples.
