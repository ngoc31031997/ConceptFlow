import { useEffect, useRef, useState } from "react";
import { useLocation } from "react-router-dom";
import styles from "./NavigationLoader.module.css";

/**
 * Shows a brief loading indicator on route changes.
 *
 * `useNavigation()` only works under a data router (`createBrowserRouter`);
 * this app uses plain `<BrowserRouter>` with statically imported pages, so
 * that hook throws "invariant" outside a data router context on every
 * render — it crashed the whole app. `useLocation` works with any router
 * and still gives a visual cue on navigation.
 */
export function NavigationLoader() {
  const location = useLocation();
  const [shouldShow, setShouldShow] = useState(false);
  const previousPath = useRef(location.pathname);

  useEffect(() => {
    if (previousPath.current === location.pathname) return;
    previousPath.current = location.pathname;

    setShouldShow(true);
    const timer = setTimeout(() => setShouldShow(false), 250);
    return () => clearTimeout(timer);
  }, [location.pathname]);

  if (!shouldShow) return null;

  return (
    <div className={styles.overlay} role="status" aria-live="polite" data-testid="navigation-loader">
      <div className={styles.spinner} aria-hidden="true" />
      <span className={styles.text}>Đang tải...</span>
    </div>
  );
}
