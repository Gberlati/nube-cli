/**
 * nube-cli OAuth broker — Cloudflare Worker
 *
 * Holds Tienda Nube app credentials server-side so CLI users don't need
 * their own credentials.json.
 *
 * Flow:
 *   1. CLI opens browser → GET /start?port=<port>
 *   2. Worker redirects  → Tienda Nube authorize page
 *   3. User authorizes   → Tienda Nube redirects to GET /callback?code=<code>&state=<port>
 *   4. Worker exchanges code for token (server-to-server)
 *   5. Worker redirects  → http://127.0.0.1:<port>/callback?token=<token>&user_id=<user_id>
 *
 * Secrets (set via `wrangler secret put`):
 *   - CLIENT_ID
 *   - CLIENT_SECRET
 */

const AUTH_BASE = "https://www.tiendanube.com/apps";
const TOKEN_URL = "https://www.tiendanube.com/apps/authorize/token";

/**
 * @param {Request} request
 * @param {{ CLIENT_ID: string, CLIENT_SECRET: string }} env
 * @returns {Promise<Response>}
 */
export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    switch (url.pathname) {
      case "/start":
        return handleStart(url, env);
      case "/callback":
        return handleCallback(url, env);
      case "/robots.txt":
        return new Response("User-agent: *\nDisallow: /\n", {
          headers: { "Content-Type": "text/plain" },
        });
      default:
        return new Response("Not Found", { status: 404 });
    }
  },
};

/**
 * GET /start?port=<port>
 *
 * Validates the port and redirects to the Tienda Nube authorization page.
 * The port is passed as `state` so the callback can redirect back to the CLI.
 */
function handleStart(url, env) {
  const port = url.searchParams.get("port");

  if (!port || !/^\d{4,5}$/.test(port)) {
    return new Response("Bad Request: invalid or missing port parameter", {
      status: 400,
    });
  }

  const authorizeURL = `${AUTH_BASE}/${env.CLIENT_ID}/authorize?state=${port}`;

  return Response.redirect(authorizeURL, 302);
}

/**
 * GET /callback?code=<code>&state=<port>
 *
 * Exchanges the authorization code for an access token, then redirects
 * the browser back to the CLI's local callback server.
 */
async function handleCallback(url, env) {
  const code = url.searchParams.get("code");
  const port = url.searchParams.get("state");

  if (!code) {
    return new Response("Bad Request: missing code parameter", { status: 400 });
  }

  if (!port || !/^\d{4,5}$/.test(port)) {
    return new Response("Bad Request: invalid or missing state parameter", {
      status: 400,
    });
  }

  const redirectURI = `${url.origin}/callback`;

  const body = new URLSearchParams({
    client_id: env.CLIENT_ID,
    client_secret: env.CLIENT_SECRET,
    grant_type: "authorization_code",
    code,
    redirect_uri: redirectURI,
  });

  let resp;

  try {
    resp = await fetch(TOKEN_URL, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body.toString(),
    });
  } catch (err) {
    return new Response(`Bad Gateway: token exchange failed: ${err.message}`, {
      status: 502,
    });
  }

  if (!resp.ok) {
    const text = await resp.text();
    return new Response(
      `Bad Gateway: token exchange returned HTTP ${resp.status}: ${text}`,
      { status: 502 },
    );
  }

  const data = await resp.json();

  if (!data.access_token) {
    return new Response("Bad Gateway: no access_token in upstream response", {
      status: 502,
    });
  }

  const callbackURL = new URL(`http://127.0.0.1:${port}/callback`);
  callbackURL.searchParams.set("token", data.access_token);

  if (data.user_id !== undefined) {
    callbackURL.searchParams.set("user_id", String(data.user_id));
  }

  return Response.redirect(callbackURL.toString(), 302);
}
