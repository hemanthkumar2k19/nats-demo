import React from 'react';
import { CoreFlowProcessor, CoreFlowProcessorProps } from './CoreFlowProcessor';

// Backward-compatible alias for CoreFlowConsumer
export type CoreFlowConsumerProps = CoreFlowProcessorProps;
export const CoreFlowConsumer: React.FC<CoreFlowProcessorProps> = (props) => {
  return <CoreFlowProcessor {...props} />;
};
