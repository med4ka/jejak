import { cookies } from "next/headers";
import NavbarClient from "./NavbarClient";

// Server Component: source of truth for the login state: the `jejak_session`
// cookie (Redis-backed through Go). MVP: checking the EXISTENCE of the cookie
// is enough; full session validation still belongs to the backend on every API
// request (there is no /api/me yet). Do NOT hardcode the login state on the
// client (the old navbar bug: the logged-in menu always appeared on the public
// landing page). The interactive part (AuthModal, avatar dropdown, hamburger)
// lives in NavbarClient ("use client"): `isLoggedIn` is passed as a prop.
export default async function Navbar() {
  const cookieStore = await cookies();
  const isLoggedIn = Boolean(cookieStore.get("jejak_session")?.value);

  return <NavbarClient isLoggedIn={isLoggedIn} />;
}
