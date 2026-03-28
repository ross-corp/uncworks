import { BrowserRouter, Routes, Route } from "react-router-dom";
import Layout from "./views/Layout";
import RunListView from "./views/RunListView";
import NewRunView from "./views/NewRunView";
import RunDetailView from "./views/RunDetailView";
import FeatureDetailView from "./views/FeatureDetailView";
import ProjectListView from "./views/ProjectListView";
import ProjectDetailView from "./views/ProjectDetailView";
import ScheduleListView from "./views/ScheduleListView";
import ScheduleDetailView from "./views/ScheduleDetailView";
import ChainListView from "./views/ChainListView";
import ChainNewView from "./views/ChainNewView";
import ChainRunListView from "./views/ChainRunListView";
import ChainRunDetailView from "./views/ChainRunDetailView";
import TemplateListView from "./views/TemplateListView";
import TemplateNewView from "./views/TemplateNewView";
import ScheduleNewView from "./views/ScheduleNewView";
import SettingsView from "./views/SettingsView";

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
          <Route path="/schedules" element={<ScheduleListView />} />
          <Route path="/schedules/:name" element={<ScheduleDetailView />} />
          <Route path="/templates" element={<TemplateListView />} />
          <Route path="/templates/new" element={<TemplateNewView />} />
          <Route path="/chains" element={<ChainListView />} />
          <Route path="/chains/new" element={<ChainNewView />} />
          <Route path="/chainruns" element={<ChainRunListView />} />
          <Route path="/schedules/new" element={<ScheduleNewView />} />
          <Route path="/chainrun/:name" element={<ChainRunDetailView />} />
          <Route path="/settings" element={<SettingsView />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
