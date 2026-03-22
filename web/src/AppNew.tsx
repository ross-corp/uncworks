import { BrowserRouter, Routes, Route } from "react-router-dom";
import Layout from "./views/Layout";
import RunListView from "./views/RunListView";
import NewRunView from "./views/NewRunView";
import RunDetailView from "./views/RunDetailView";
import FeatureDetailView from "./views/FeatureDetailView";
import ProjectListView from "./views/ProjectListView";
import ProjectDetailView from "./views/ProjectDetailView";

/**
 * App shell — views with URL routing.
 */
export default function AppNew() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<RunListView />} />
          <Route path="/new" element={<NewRunView />} />
          <Route path="/run/:id" element={<RunDetailView />} />
          <Route path="/feature/:name" element={<FeatureDetailView />} />
          <Route path="/projects" element={<ProjectListView />} />
          <Route path="/projects/:name" element={<ProjectDetailView />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
