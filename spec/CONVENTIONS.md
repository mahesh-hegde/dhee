## Coding guidelines and conventions
* Rely on static typing and tooling
  * Do not assume anything about fields/members of types which are not provided. Ask to provide them instead. Static typing exists for a reason.

* Be concise and simple:
  * Always provide a short docstring for public methods. But do not be verbose. Follow go/dart commenting style which is short and only describes the behavior of parameters which is not already described by types.
  * Do not use UNICODE characters in comments, but you can use them in string literals if the use case requires it.

* Reliability is important.
  * Do not swallow errors.
  * Reduce usage of `any` type and type casting.
  * Do not ignore errors. Return error gracefully if caller expects it / in request-response lifecycle. Fail fast in global init / one-time jobs where error return is not possible.

* Correct me when I am wrong:
  * If any instructions I provide are suboptimal, or you can do the same thing in better way, stop implementing and ask me for confirmation, mentioning the better way. I am a very humble person.
