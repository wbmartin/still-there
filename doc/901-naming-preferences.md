# Code Naming Preferences

- when you name a list variable, do not use the plural form. instead suffix it with \_list or List (depending on the appropriate naming convention in use)
- prioritize readability when naming a variable - choose more verbose names like "tagCache" instead of "tc"
- do not shorten variables that implement a class name e.g. a variable implementing the ShrmplKVClient should be shrmplKVClient; not shrimplKV, not shrmplKVC.  One exception is the Golang method receiver - it can be shortened to 1 char.
