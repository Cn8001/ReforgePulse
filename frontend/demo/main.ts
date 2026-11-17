// The transport has to be in place before the app mounts and makes its first
// call, so it is set up in its own module and imported first.
import "./transport";
import "../src/main";
