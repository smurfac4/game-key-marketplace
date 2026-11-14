import { usePreferences } from "./content/preferences";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createBrowserRouter, RouterProvider, Link } from "react-router-dom";
import { AuthProvider, Protected } from "./auth/AuthContext";
import { Layout } from "./layout/Layout";
import { Homepage } from "./home/Homepage";
import { Store } from "./pages/Store";
import { ProductPage } from "./pages/ProductPage";
import {
  AccountPage,
  AccountValuePage,
  WishlistPage,
  KeysLibrary,
} from "./pages/AccountPages";
import { AuthPage } from "./pages/AuthPage";
import { CartPage } from "./pages/CartPage";
import { CheckoutPage } from "./pages/CheckoutPage";
import { OrderHistory, OrderPage } from "./pages/Orders";
import { EmptyState } from "./components/ui";
import { copy } from "./content/copy";
import "./styles.css";
const queryClient = new QueryClient({
  defaultOptions: { queries: { refetchOnWindowFocus: false, retry: 1 } },
});
function Help() {
  return (
    <div className="container-page max-w-3xl">
      <h1 className="page-title">{copy.help}</h1>
      <p className="mb-8 text-lg">{copy.helpIntro}</p>
      <ol className="list-inside list-decimal space-y-4">
        {copy.helpSteps.map((step) => (
          <li key={step} className="panel">
            {step}
          </li>
        ))}
      </ol>
      <p className="mt-8 leading-relaxed text-muted">{copy.helpNote}</p>
      <Link to="/store" className="button mt-6">
        {copy.browse}
      </Link>
    </div>
  );
}
function StatusPage({ error = false }: { error?: boolean }) {
  return (
    <div className="container-page">
      <EmptyState title={error ? copy.error : copy.notFound}>
        <a className="button" href="/">
          {copy.returnHome}
        </a>
      </EmptyState>
    </div>
  );
}
const router = createBrowserRouter([
  {
    element: <Layout />,
    errorElement: <StatusPage error />,
    children: [
      { path: "/", element: <Homepage /> },
      { path: "/store", element: <Store /> },
      { path: "/help", element: <Help /> },
      { path: "/search", element: <Store /> },
      { path: "/products/:slug", element: <ProductPage /> },
      { path: "/login", element: <AuthPage /> },
      { path: "/register", element: <AuthPage register /> },
      {
        path: "/account",
        element: (
          <Protected>
            <AccountPage />
          </Protected>
        ),
      },
      {
        path: "/wishlist",
        element: (
          <Protected>
            <WishlistPage />
          </Protected>
        ),
      },
      {
        path: "/library",
        element: (
          <Protected>
            <KeysLibrary />
          </Protected>
        ),
      },
      ...(["balance", "rewards", "cashback"] as const).map((kind) => ({
        path: `/account/${kind}`,
        element: (
          <Protected>
            <AccountValuePage kind={kind} />
          </Protected>
        ),
      })),
      {
        path: "/cart",
        element: (
          <Protected>
            <CartPage />
          </Protected>
        ),
      },
      {
        path: "/checkout",
        element: (
          <Protected>
            <CheckoutPage />
          </Protected>
        ),
      },
      {
        path: "/orders",
        element: (
          <Protected>
            <OrderHistory />
          </Protected>
        ),
      },
      {
        path: "/orders/:id",
        element: (
          <Protected>
            <OrderPage />
          </Protected>
        ),
      },
      {
        path: "*",
        element: <StatusPage />,
      },
    ],
  },
]);
function App() {
  const prefs = usePreferences();
  return (
    <AuthProvider key={prefs.language + prefs.currency}>
      <RouterProvider router={router} />
    </AuthProvider>
  );
}
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
