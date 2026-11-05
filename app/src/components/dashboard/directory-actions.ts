"use server";

import { getDirectory, type Directory } from "@/lib/directory";

/** Lets client pickers load names without pulling session code into the browser. */
export async function loadDirectory(): Promise<Directory> {
  return getDirectory();
}
