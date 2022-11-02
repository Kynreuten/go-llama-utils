# RegEx Breakdowns


## Parse Environment variables from .env files
We want to parse any valid lines into Key / Value pairs. 


### First seemingly working attempt
`^[ \t]*(?:export)?[ \t]*([A-Z][\w-]*)=((?:\"?(?:[^\r\n\$\"]*(?:\$\{?[A-Z][\w-]*\}?)*)+\"?)|(?:[\.\w\-:\/\\]*(?:\${[A-Z][\w-]*\})*)+){1}$`
OR
```
^[ \t]*(?:export)?[ \t]*
(
    [A-Z][\w-]*)=
    (
        (?:\"?(?:[^\r\n\$\"]*(?:\$\{?[A-Z][\w-]*\}?)*)+\"?)
        |(?:[\.\w\-:\/\\]*(?:\${[A-Z][\w-]*\})*
    )+
){1}$
```

#### Breakdown

##### Tiny Pieces
**Valid Variable**
`[A-Z][\w-]*`
Starts with uppercase letter then can be any valid word character as well as dash ('-').

##### Start
`[ \t]*(?:export)?[ \t]?`
Allow for preceding whitespace (but only space or tab) and an optional 'export' declaration (which we ignore) with more whitespace allowed.

##### Key
`([A-Z][\w-]*)=`
Capture group for the "Key" that allows any number of variable-valid characters (must start with a letter). Ends when we find the '='

##### Value
```
(
    ...
)$
```
This whole section is defined by the surrounding group that requires it end at the end of the line. Allows us to capture the entire value as one group (including any quotes and such)

**Double-Quoted Value surround**
`(?:[\"\']?...[\"\']?)`
Non-capturing group that looks for the value being surrounded by double quotes.

(?:
    `[^\r\n\$\"]*`
    Look for almost anything. Just can't be a variable or the end of the value. None required however, could go to the end of the value.

    `(?:\$[\{]?[A-Z][\w-]*[\}]?)*`
    Variable declaration. Either like "${VALID_VAR}" or "$VALID_VAR". None required. Could go to the end of the value.
)+
Allow one or more of these between the outer quotes


**TODO: Single-Quoted Value surround**
Should allow whatever characters desired. There's no variable expansion in here!

**Non-Quoted Value**
`|(?:[\.\w\-:\/\\]*(?:\${[A-Z][\w-]*\})*)+`
Here we should be looking for similar values, but with no surrounding quotes. No quotes does mean that whitespace is not allowed however! The pipe '|' indicates that this is an OR to the previous quoted option.


##TODO: Handle single quoted values
