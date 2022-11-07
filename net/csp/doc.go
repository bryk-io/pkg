/*
Package csp provides an easy-to-use "Content Security Policy" implementation.

Content Security Policy (CSP) is an added layer of security that helps to detect
and mitigate certain types of attacks, including Cross-Site Scripting (XSS) and
data injection attacks. These attacks are used for everything from data theft, to
site defacement, to malware distribution.

As a general rule, a majority of complex web applications are susceptible to XSS,
and would benefit from adopting CSP. In particular, CSP is recommended for applications
which manage sensitive data such as administrative UIs and device management consoles,
or products hosting user-generated documents, messages or media files. Especially
in products using modern frameworks (Closure Templates) adopting CSP can be
relatively straightforward and provide a large security improvement in exchange
for a small-time investment.

To enable a strict CSP policy (preventing the execution of untrusted scripts),
most applications will need to make the following changes:
  - Add a nonce attribute to all <script> elements. Some template systems can do
    this automatically.
  - Refactor any markup with inline event handlers (onclick, etc.) and `javascript:`
    URIs .
  - For every page load, generate a new nonce, pass it the to the template system,
    and use the same value in the policy.

To add the nonce to a `<script>` tag use:

	<script nonce="{nonce}" src="/path/to/script.js"></script>

More information:
https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
*/
package csp
