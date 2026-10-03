"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useTranslation } from "../../lib/I18nProvider";

// The /links page is DEPRECATED: the old version exposed ALL short links from
// EVERY creator to the public without an ownership filter (a minor privacy
// leak). The Profile Dashboard already covers viewing one's own links and
// analytics. This route is kept only as a redirect entry point so old bookmarks
// do not 404: logged in -> /dashboard, not logged in -> /.
export default function LinksRedirect() {
  const router = useRouter();
  const { t } = useTranslation();

  useEffect(() => {
    const username = localStorage.getItem("jejak_username");
    router.replace(username ? "/dashboard" : "/");
  }, [router]);

  return (
    <div className="mx-auto w-full max-w-2xl px-4 pb-16 pt-24">
      <main>
        <p className="text-sm text-muted">{t("dashboard.redirectNotice")}</p>
      </main>
    </div>
  );
}
