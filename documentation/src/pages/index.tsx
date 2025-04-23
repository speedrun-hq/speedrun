import {Redirect} from '@docusaurus/router';
import React from 'react';

export default function Home(): React.ReactElement {
  return <Redirect to="/intro" />;
}
