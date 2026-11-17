import { ref } from "vue";

// GitHub Pages has no rewrites, so pages live in the hash: #/features/cleanup.
export type Page = "home" | "features" | "how" | "install";
const pages: Page[] = ["home", "features", "how", "install"];

export interface Route {
  page: Page;
  sub: string;
}

function parse(hash: string): Route {
  const [page = "", sub = ""] = hash.replace(/^#\/?/, "").split("/");
  return { page: pages.includes(page as Page) ? (page as Page) : "home", sub };
}

export const route = ref<Route>(parse(location.hash));

window.addEventListener("hashchange", () => {
  route.value = parse(location.hash);
  window.scrollTo({ top: 0 });
});

export const href = (page: Page, sub = "") => (page === "home" ? "#/" : `#/${page}${sub ? `/${sub}` : ""}`);
