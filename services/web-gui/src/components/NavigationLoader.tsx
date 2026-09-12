import { useEffect, useState } from "react";
import { useNavigation } from "react-router-dom";
import styles from "./NavigationLoader.module.css";

/**
 * Shows a loading indicator when React Router is navigating between pages.
 * Uses the router's native navigation state to avoid manual coordination.
 */
export function NavigationLoader() {
  const navigation = useNavigation();
  const [shouldShow, setShouldShow] = useState(false);

  useEffect(() => {
    if (navigation.state === "loading") {
      // Delay showing the loader to avoid flashing for fast navigations
      const timer = setTimeout(() => setShouldShow(true), 100);
      return () => clearTimeout(timer);
    } else {
      setShouldShow(false);
    }
  }, [navigation.state]);

  if (!shouldShow) return null;

  return (
    <div className={styles.overlay} role="status" aria-live="polite" data-testid="navigation-loader">
      <div className={styles.spinner} aria-hidden="true" />
      <span className={styles.text}>Đang tải...</span>
    </div>
  );
}
