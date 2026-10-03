"use client";

import { usePathname } from "next/navigation";
import { AnimatePresence, motion } from "framer-motion";
import { EASE } from "./animations";

// Global page transition: wrapped by layout.jsx (see Apple detail #5).
// Entering a new page: opacity 0 → 1 + y 8 → 0, 250ms EASE (transform +
// opacity only: no layout-property animation).
// AnimatePresence mode="wait" + initial={false}: first load WITHOUT
// animation (no content flash), page change = 250ms enter without the old
// exit (immediate swap, smooth fade-in: the old exit is deliberately left
// undefined so navigation never feels like it "waits" ~450ms).
// Located in lib/ (not components/) per the spec contract; the old file
// app/components/PageTransition.jsx (which turned out to be unused: a dead
// import in layout) was removed and moved here.
export default function PageTransition({ children }) {
  const pathname = usePathname();
  return (
    <AnimatePresence mode="wait" initial={false}>
      <motion.div
        key={pathname}
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.25, ease: EASE }}
      >
        {children}
      </motion.div>
    </AnimatePresence>
  );
}
