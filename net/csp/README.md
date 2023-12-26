# Package `csp`

Content Security Policy (CSP) is an added layer of security that helps to detect
and mitigate certain types of attacks, including Cross-Site Scripting (XSS) and
data injection attacks. These attacks are used for everything from data theft, to
site defacement and malware distribution.

As a general rule, a majority of complex web applications are susceptible to XSS,
and would benefit from adopting CSP. In particular, CSP is recommended for applications
which manage sensitive data such as administrative UIs and device management consoles,
or products hosting user-generated documents, messages or media files. Especially in
products using modern frameworks (Closure Templates) adopting CSP can be relatively
straightforward and provide a large security improvement in exchange for a small-time
investment.

More information: <https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP>

To enable a strict CSP policy (preventing the execution of untrusted scripts),
most applications will need to make the following changes:

- For every page load, the server generates a new `nonce`.
- Add the nonce attribute to all `<script>` elements.
- The `nonce` is a random value that is unique to each page load; used
  to mark scripts that are allowed to execute on the page.

To add the nonce to a `<script>` tag use:

```html
<script nonce="{nonce}" src="/path/to/script.js"></script>
```

## Usage

First you need to define a CSP policy.

```go
options := []Option{
  // disables <base> URIs, preventing attackers from changing the locations of scripts
  // loaded from relative URLs
  WithBaseURI("'none'"),
  // report policy violations
  WithReportTo("/reports", "/another-endpoint"),
  // disable loading all external content
  WithDefaultSrc("'self'"),
  // don't enforce policy; use only for testing
  WithReportOnly(),
  // loose JS execution restrictions; use only for testing
  UnsafeEval(),
}

// Create your policy object
policy, _ := New(options...)

// the server can enforce the policy by using
// `policy.Handler` as a middleware.
```

You can then use the policy to generate a nonce for each request.

```go
nonce := policy.Refresh()
```
