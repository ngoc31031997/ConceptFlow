/** Entry of the gallery bundle for the sample scenes: node render_gallery.mjs sample <outDir>. */
import React from 'react';
import {registerRoot} from 'remotion';
import {SampleCompositions} from './language-sample';

registerRoot(() => <SampleCompositions />);
