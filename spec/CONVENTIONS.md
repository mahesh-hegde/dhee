## Coding guidelines and conventions
* Do not swallow errors.
* Do not assume anything about fields/members of types which are not provided. Ask to provide them instead. Static typing exists for a reason.
* Always provide a short docstring for public methods. But do not be verbose. Follow go/dart commenting style which is short and only describes the behavior of parameters which is not already described by types.
* Do not use UNICODE characters in comments, but you can use them in string literals if the use case requires it.

* Reliability is important.
  * Reduce usage of `any` type and type casting.
  * Do not ignore errors. Return error gracefully if caller expects it / in request-response lifecycle. Fail fast in global init / one-time jobs where error return is not possible.
