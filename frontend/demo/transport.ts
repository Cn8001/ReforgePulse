import { setTransport, objectNames } from "@wailsio/runtime";
// @ts-ignore: provided by the demo build's plugin
import ids from "virtual:binding-ids";
// @ts-ignore: plain data module
import { backend } from "./backend.js";

const names = new Map<number, string>(Object.entries(ids as Record<string, number>).map(([name, id]) => [id, name]));

setTransport({
  // Window controls, events and dialogs have nothing to act on in a page, so
  // only service calls get an answer.
  async call(objectID: number, _method: number, _window: string, args: any) {
    if (objectID !== objectNames.Call) return null;
    const handler = backend[names.get(args?.methodID) ?? ""];
    return handler ? handler(...(args?.args ?? [])) : null;
  },
});
