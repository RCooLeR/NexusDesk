import {Card} from '../../components/ui';
import type {ToolEvent} from '../../types';

type ToolTimelineProps = {
    events: ToolEvent[];
};

export function ToolTimeline({events}: ToolTimelineProps) {
    return (
        <Card className="timeline">
            <div className="pane-title">
                <span>Activity Log</span>
                <small>Model, tools, and workspace events</small>
            </div>
            <div className="timeline-items">
                {events.map((event, index) => (
                    <div className="timeline-item" key={`${event.time}-${event.title}-${index}`}>
                        <time>{event.time}</time>
                        <strong>{event.title}</strong>
                        <p>{event.detail}</p>
                    </div>
                ))}
            </div>
        </Card>
    );
}
