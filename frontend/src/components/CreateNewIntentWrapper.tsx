"use client";

import dynamic from "next/dynamic";

const CreateNewIntent = dynamic(() => import("./CreateNewIntent"), {
  ssr: false,
});

export function CreateNewIntentWrapper() {
  return <CreateNewIntent />;
}
