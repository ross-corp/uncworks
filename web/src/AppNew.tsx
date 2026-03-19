import { BrowserRouter, Routes, Route } from "react-router-dom";
import Layout from "./views/Layout";
import RunListView from "./views/RunListView";
import NewRunView from "./views/NewRunView";
import RunDetailView from "./views/RunDetailView";

/**
 * New app shell — three views with URL routing.
 * Replaces the old dashboard App.tsx.
 */
export default function AppNew() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<RunListView />} />
          <Route path="/new" element={<NewRunView />} />
          <Route path="/run/:id" element={<RunDetailView />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
